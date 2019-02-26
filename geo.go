package main

import (
	"fmt"
	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"
	"log"
	"net/url"
	"time"
	"tinderFetch/api"
)

// EarthRadiusM Earth radius in meters
const (
	EarthRadiusM    = 6371000.0
	DefaultMaxLevel = 13
	DefaultMinLevel = 2
	DefaultMaxCells = 1024
	MilesInKm       = 1.60934
)

func getSidewalkURL(reqCoord api.Coordinates, cu s2.CellUnion) string {
	v := url.Values{}
	v.Set("center", fmt.Sprintf("%f,%f", reqCoord.Latitude, reqCoord.Longitude))
	v.Set("zoom", "15")
	var cellTokens string
	for _, cellID := range cu {
		if len(cellTokens) < 1 {
			cellTokens = cellID.ToToken()
		}
		cellTokens = fmt.Sprintf("%s,%s", cellTokens, cellID.ToToken())
	}
	v.Set("cells", cellTokens)
	return "https://s2.sidewalklabs.com/regioncoverer/?" + v.Encode()
}

func getAngleByRadius(radius float64) s1.Angle {
	centerAngle := float64(radius) / EarthRadiusM
	return s1.Angle(centerAngle)
}

func getRadiusByAngle(angle s1.Angle) float64 {
	return EarthRadiusM * angle.Radians()
}

func getEdgeCellsByRadius(cu s2.CellUnion, reqCoord api.Coordinates, radius float64) s2.CellUnion {
	ll := s2.LatLngFromDegrees(reqCoord.Latitude, reqCoord.Longitude)
	centerPoint := s2.PointFromLatLng(ll)
	var edgeCells s2.CellUnion
	for _, cellID := range cu {
		cell := s2.CellFromCellID(cellID)
		distance := getRadiusByAngle(cell.MaxDistance(centerPoint).Angle())
		if distance >= float64(radius) {
			edgeCells = append(edgeCells, cellID)
		}
	}
	return edgeCells
}

// take the smallest cell from union
func getRandomPosition(cu s2.CellUnion) api.Coordinates {
	var chosenPosition api.Coordinates
	var maxLevel int
	for _, cellID := range cu {
		if cellID.Level() > maxLevel {
			maxLevel = cellID.Level()
			ll := cellID.LatLng()
			chosenPosition = api.Coordinates{Latitude: ll.Lat.Degrees(), Longitude: ll.Lng.Degrees()}
		}
	}
	return chosenPosition
}

func milesToMeters(mi int) float64 {
	return float64(mi) * MilesInKm * 1000
}

func getCellsByCoord(reqCoord api.Coordinates, radius float64) s2.CellUnion {
	ll := s2.LatLngFromDegrees(reqCoord.Latitude, reqCoord.Longitude)
	point := s2.PointFromLatLng(ll)
	cap := s2.CapFromCenterAngle(point, getAngleByRadius(radius))
	rc := s2.RegionCoverer{MaxLevel: DefaultMaxLevel, MinLevel: DefaultMinLevel, MaxCells: DefaultMaxCells}
	return rc.Covering(cap)
}

func getUserPosition(tinderAPI *api.TinderAPI, userID string, profilePos api.Coordinates, tiles s2.CellUnion) (api.Coordinates, s2.CellUnion, error) {
	user, err := tinderAPI.GetUser(userID)
	if err != nil {
		return profilePos, nil, err
	}
	distance := milesToMeters(user.Distance)
	log.Printf("Distance between you and user: %f meters", distance)
	oldTiles := tiles
	if user.Distance <= 1 {
		log.Println("Distance too short. Return cell related to current position")
		tiles = getCellsByCoord(profilePos, distance)
		if oldTiles != nil {
			tiles = s2.CellUnionFromIntersection(oldTiles, tiles)
		}
		log.Println(getSidewalkURL(profilePos, tiles))
		return profilePos, tiles, nil
	}
	tiles = getEdgeCellsByRadius(getCellsByCoord(profilePos, distance), profilePos, distance)
	if oldTiles != nil {
		log.Printf("Make intersection between two tile sets")
		tiles = s2.CellUnionFromIntersection(oldTiles, getEdgeCellsByRadius(tiles, profilePos, distance))
	}
	log.Println(getSidewalkURL(profilePos, tiles))
	if len(tiles) < 1 {
		log.Println("No intersected tiles. Something went wrong. Return last non-empty tile set")
		return profilePos, oldTiles, nil
	}
	if len(tiles) <= 1 {
		log.Println("Only one tile, return it")
		if tiles[0].Level() < DefaultMaxLevel {
			log.Println("Tile are too big. Split it")
			children := tiles[0].Children()
			return getUserPosition(tinderAPI, userID, profilePos, children[:])
		}
		return profilePos, tiles, nil
	}
	newCoord := getRandomPosition(tiles)
	log.Printf("Change user position to %v (sleep for 5 minutes)", newCoord)
	time.Sleep(time.Minute * 5)
	if err := tinderAPI.ChangeProfilePosition(newCoord); err != nil {
		log.Printf("Unable to change user position: %v", err)
		return profilePos, tiles, nil
	}
	return getUserPosition(tinderAPI, userID, newCoord, tiles)
}

// GetUserPosition get user approximate geo position by user ID. It doesn't restore your position
func GetUserPosition(tinderAPI *api.TinderAPI, userID string) (string, error) {
	profile, err := tinderAPI.GetProfile()
	if err != nil {
		return "", err
	}
	/*
		currentPosition := profile.Position
		defer func() {
			if err := api.ChangeProfilePosition(currentPosition); err != nil {
				log.Printf("Unable to restore user position: %v", err)
			}
		}()
	*/
	newPosition, tiles, err := getUserPosition(tinderAPI, userID, profile.Position, nil)
	if err != nil {
		return "", err
	}
	return getSidewalkURL(newPosition, tiles), nil
}

package main

type Config struct {
	Area Config_Area `yaml:"area"`
}

type Config_Area struct {
	Coords []*Coord `yaml:"coords"`
}

type Coord struct {
	Latitude  float64 `yaml:"latitude"`
	Longitude float64 `yaml:"longitude"`
}

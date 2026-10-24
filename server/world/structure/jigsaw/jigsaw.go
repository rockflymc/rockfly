package jigsaw

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// JointType represents whether connected jigsaw pieces can roll or must align rigidly.
type JointType int

const (
	JointRollable JointType = iota
	JointAligned
)

// JigsawBlock models a Jigsaw block inside a structure template that defines connection anchors.
type JigsawBlock struct {
	Pos        cube.Pos
	Name       string
	Target     string
	Pool       string
	Facing     cube.Face
	Joint      JointType
	FinalBlock world.Block
}

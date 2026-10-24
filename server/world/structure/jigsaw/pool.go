package jigsaw

// WeightedPiece assigns a relative selection weight to a candidate structure piece.
type WeightedPiece struct {
	Piece  *Piece
	Weight int
}

// Pool represents a named collection of candidate pieces and a fallback pool for dead ends.
type Pool struct {
	Name        string
	Pieces      []WeightedPiece
	Fallback    string
	totalWeight int
}

// NewPool constructs a new structure pool from weighted pieces.
func NewPool(name, fallback string, pieces ...WeightedPiece) *Pool {
	total := 0
	for _, p := range pieces {
		if p.Weight > 0 {
			total += p.Weight
		}
	}
	return &Pool{
		Name:        name,
		Pieces:      pieces,
		Fallback:    fallback,
		totalWeight: total,
	}
}

// Select deterministically picks a piece from the pool based on a pseudo-random integer.
func (p *Pool) Select(randVal int) *Piece {
	if len(p.Pieces) == 0 {
		return nil
	}
	if p.totalWeight <= 0 {
		return p.Pieces[0].Piece
	}

	choice := (randVal%p.totalWeight + p.totalWeight) % p.totalWeight
	accum := 0
	for _, wp := range p.Pieces {
		accum += wp.Weight
		if choice < accum {
			return wp.Piece
		}
	}
	return p.Pieces[len(p.Pieces)-1].Piece
}

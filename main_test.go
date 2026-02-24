package main

import (
	"testing"
)

func TestInitLevelNoOverlaps(t *testing.T) {
	g := &Game{Level: 1}
	
	// Run multiple times to increase confidence (due to randomness)
	for run := 0; run < 100; run++ {
		g.initLevel()
		
		// Check berries vs walls
		for _, berry := range g.Berries {
			for _, wall := range g.Walls {
				if berry.X == wall.X && berry.Y == wall.Y {
					t.Errorf("Berry and Wall overlap at %v", berry)
				}
			}
			
			// Check berry vs player
			if berry.X == g.Player.X && berry.Y == g.Player.Y {
				t.Errorf("Berry and Player overlap at %v", berry)
			}
		}
		
		// Check walls vs player
		for _, wall := range g.Walls {
			if wall.X == g.Player.X && wall.Y == g.Player.Y {
				t.Errorf("Wall and Player overlap at %v", wall)
			}
		}
		
		// Check enemies vs player
		for _, enemy := range g.Enemies {
			if enemy.Pos.X == g.Player.X && enemy.Pos.Y == g.Player.Y {
				t.Errorf("Enemy and Player overlap at %v", enemy.Pos)
			}
		}
	}
}

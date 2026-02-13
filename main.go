package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"golang.org/x/term"
)

const (
	Width  = 40
	Height = 20
)

type Position struct {
	X, Y int
}

type Enemy struct {
	Pos       Position
	Direction int // 0=up, 1=right, 2=down, 3=left
}

type Game struct {
	Player        Position
	Berries       []Position
	Enemies       []Enemy
	Walls         []Position
	Score         int
	Level         int
	BerriesNeeded int
	GameOver      bool
	Won           bool
}

func clearScreen() {
	// ANSI escape: move cursor to top-left and clear screen
	fmt.Print("\033[H\033[2J")
}

// print outputs a line with \r\n for raw terminal mode
func printLn(s string) {
	fmt.Print(s + "\r\n")
}

func (g *Game) initLevel() {
	g.Player = Position{Width / 2, Height / 2}
	g.Berries = []Position{}
	g.Enemies = []Enemy{}
	g.Walls = []Position{}
	g.BerriesNeeded = 5 + g.Level*2

	// Generate berries
	for i := 0; i < g.BerriesNeeded; i++ {
		g.Berries = append(g.Berries, Position{
			X: rand.Intn(Width-2) + 1,
			Y: rand.Intn(Height-2) + 1,
		})
	}

	// Generate enemies (increases with level)
	numEnemies := 1 + g.Level/2
	for i := 0; i < numEnemies; i++ {
		g.Enemies = append(g.Enemies, Enemy{
			Pos: Position{
				X: rand.Intn(Width-2) + 1,
				Y: rand.Intn(Height-2) + 1,
			},
			Direction: rand.Intn(4),
		})
	}

	// Generate some random walls
	numWalls := 10 + g.Level*3
	for i := 0; i < numWalls; i++ {
		g.Walls = append(g.Walls, Position{
			X: rand.Intn(Width-2) + 1,
			Y: rand.Intn(Height-2) + 1,
		})
	}
}

func (g *Game) draw() {
	clearScreen()

	// Create grid
	grid := make([][]rune, Height)
	for i := range grid {
		grid[i] = make([]rune, Width)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	// Draw borders
	for x := 0; x < Width; x++ {
		grid[0][x] = '═'
		grid[Height-1][x] = '═'
	}
	for y := 0; y < Height; y++ {
		grid[y][0] = '║'
		grid[y][Width-1] = '║'
	}
	grid[0][0] = '╔'
	grid[0][Width-1] = '╗'
	grid[Height-1][0] = '╚'
	grid[Height-1][Width-1] = '╝'

	// Draw walls
	for _, wall := range g.Walls {
		if wall.X > 0 && wall.X < Width-1 && wall.Y > 0 && wall.Y < Height-1 {
			grid[wall.Y][wall.X] = '█'
		}
	}

	// Draw berries
	for _, berry := range g.Berries {
		if berry.X > 0 && berry.X < Width-1 && berry.Y > 0 && berry.Y < Height-1 {
			grid[berry.Y][berry.X] = '●'
		}
	}

	// Draw enemies
	for _, enemy := range g.Enemies {
		if enemy.Pos.X > 0 && enemy.Pos.X < Width-1 && enemy.Pos.Y > 0 && enemy.Pos.Y < Height-1 {
			grid[enemy.Pos.Y][enemy.Pos.X] = '☠'
		}
	}

	// Draw player
	if g.Player.X > 0 && g.Player.X < Width-1 && g.Player.Y > 0 && g.Player.Y < Height-1 {
		grid[g.Player.Y][g.Player.X] = '◆'
	}

	// Print grid
	for _, row := range grid {
		printLn(string(row))
	}

	// Print stats
	printLn("")
	printLn("╔═══════════════════════════════════════╗")
	printLn(fmt.Sprintf("║ Level: %-3d  Score: %-6d  Berries: %d/%d ║",
		g.Level, g.Score, g.BerriesNeeded-len(g.Berries), g.BerriesNeeded))
	printLn("╚═══════════════════════════════════════╝")
	printLn("")
	printLn("Controls: W=Up, S=Down, A=Left, D=Right, Q=Quit")

	if g.GameOver {
		printLn("")
		printLn("*** GAME OVER! You were caught by an alien! ***")
	}
	if g.Won {
		printLn("")
		printLn("*** LEVEL COMPLETE! Press any key for next level ***")
	}
}

func (g *Game) movePlayer(dx, dy int) {
	newX := g.Player.X + dx
	newY := g.Player.Y + dy

	// Check boundaries
	if newX <= 0 || newX >= Width-1 || newY <= 0 || newY >= Height-1 {
		return
	}

	// Check walls
	for _, wall := range g.Walls {
		if wall.X == newX && wall.Y == newY {
			return
		}
	}

	g.Player.X = newX
	g.Player.Y = newY

	// Check berry collection
	for i, berry := range g.Berries {
		if berry.X == g.Player.X && berry.Y == g.Player.Y {
			g.Berries = append(g.Berries[:i], g.Berries[i+1:]...)
			g.Score += 10
			break
		}
	}

	// Check if level complete
	if len(g.Berries) == 0 {
		g.Won = true
	}
}

func (g *Game) moveEnemies() {
	for i := range g.Enemies {
		enemy := &g.Enemies[i]

		// Sometimes chase player
		if rand.Float32() < 0.3 {
			if g.Player.X > enemy.Pos.X {
				enemy.Direction = 1 // right
			} else if g.Player.X < enemy.Pos.X {
				enemy.Direction = 3 // left
			} else if g.Player.Y > enemy.Pos.Y {
				enemy.Direction = 2 // down
			} else if g.Player.Y < enemy.Pos.Y {
				enemy.Direction = 0 // up
			}
		}

		// Move in current direction
		dx, dy := 0, 0
		switch enemy.Direction {
		case 0:
			dy = -1
		case 1:
			dx = 1
		case 2:
			dy = 1
		case 3:
			dx = -1
		}

		newX := enemy.Pos.X + dx
		newY := enemy.Pos.Y + dy

		// Check if valid move
		valid := true
		if newX <= 0 || newX >= Width-1 || newY <= 0 || newY >= Height-1 {
			valid = false
		}

		// Check walls
		for _, wall := range g.Walls {
			if wall.X == newX && wall.Y == newY {
				valid = false
				break
			}
		}

		if valid {
			enemy.Pos.X = newX
			enemy.Pos.Y = newY
		} else {
			// Change direction if blocked
			enemy.Direction = rand.Intn(4)
		}

		// Check collision with player
		if enemy.Pos.X == g.Player.X && enemy.Pos.Y == g.Player.Y {
			g.GameOver = true
		}
	}
}

func getInput() (rune, error) {
	var b [1]byte
	_, err := os.Stdin.Read(b[:])
	return rune(b[0]), err
}

func main() {
	// Put terminal into raw mode for immediate keypress reading
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set raw mode: %v\n", err)
		os.Exit(1)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	rand.Seed(time.Now().UnixNano())

	game := &Game{
		Level: 1,
	}
	game.initLevel()

	// Game loop
	inputChan := make(chan rune)
	go func() {
		for {
			ch, err := getInput()
			if err == nil {
				inputChan <- ch
			}
		}
	}()

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	game.draw()

	for !game.GameOver {
		select {
		case <-ticker.C:
			if !game.Won {
				game.moveEnemies()
				game.draw()
			}

		case input := <-inputChan:
			if game.Won {
				game.Level++
				game.Won = false
				game.initLevel()
				game.draw()
				continue
			}

			switch input {
			case 'w', 'W':
				game.movePlayer(0, -1)
			case 's', 'S':
				game.movePlayer(0, 1)
			case 'a', 'A':
				game.movePlayer(-1, 0)
			case 'd', 'D':
				game.movePlayer(1, 0)
			case 'q', 'Q':
				printLn("\r\nThanks for playing XZAP!")
				return
			}
			game.draw()
		}
	}

	game.draw()
	printLn(fmt.Sprintf("\r\nFinal Score: %d", game.Score))
	printLn("Thanks for playing XZAP!")
}

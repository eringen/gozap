#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <termios.h>
#include <unistd.h>
#include <time.h>
#include <fcntl.h>
#include <poll.h>

#define WIDTH 40
#define HEIGHT 20
#define MAX_BERRIES 64
#define MAX_ENEMIES 16
#define MAX_WALLS 128

typedef struct { int x, y; } Pos;
typedef struct { Pos pos; int dir; } Enemy;

typedef struct {
    Pos player;
    Pos berries[MAX_BERRIES];
    int num_berries;
    Enemy enemies[MAX_ENEMIES];
    int num_enemies;
    Pos walls[MAX_WALLS];
    int num_walls;
    int score;
    int level;
    int berries_needed;
    int game_over;
    int won;
} Game;

static struct termios orig_termios;

static void restore_terminal(void) {
    tcsetattr(STDIN_FILENO, TCSAFLUSH, &orig_termios);
}

static void raw_mode(void) {
    tcgetattr(STDIN_FILENO, &orig_termios);
    atexit(restore_terminal);
    struct termios raw = orig_termios;
    raw.c_lflag &= ~(ECHO | ICANON);
    raw.c_cc[VMIN] = 0;
    raw.c_cc[VTIME] = 0;
    tcsetattr(STDIN_FILENO, TCSAFLUSH, &raw);
}

static int is_wall(Game *g, int x, int y) {
    for (int i = 0; i < g->num_walls; i++)
        if (g->walls[i].x == x && g->walls[i].y == y) return 1;
    return 0;
}

static void init_level(Game *g) {
    g->player = (Pos){WIDTH / 2, HEIGHT / 2};
    g->berries_needed = 5 + g->level * 2;
    if (g->berries_needed > MAX_BERRIES) g->berries_needed = MAX_BERRIES;
    g->num_berries = g->berries_needed;
    g->num_enemies = 1 + g->level / 2;
    if (g->num_enemies > MAX_ENEMIES) g->num_enemies = MAX_ENEMIES;
    g->num_walls = 10 + g->level * 3;
    if (g->num_walls > MAX_WALLS) g->num_walls = MAX_WALLS;
    g->won = 0;
    g->game_over = 0;

    for (int i = 0; i < g->num_berries; i++)
        g->berries[i] = (Pos){rand() % (WIDTH - 2) + 1, rand() % (HEIGHT - 2) + 1};
    for (int i = 0; i < g->num_enemies; i++)
        g->enemies[i] = (Enemy){{rand() % (WIDTH - 2) + 1, rand() % (HEIGHT - 2) + 1}, rand() % 4};
    for (int i = 0; i < g->num_walls; i++)
        g->walls[i] = (Pos){rand() % (WIDTH - 2) + 1, rand() % (HEIGHT - 2) + 1};
}

static void draw(Game *g) {
    /* Grid uses single chars: '#' for walls, 'o' for berries, 'X' for enemies, '@' for player */
    char grid[HEIGHT][WIDTH + 1];

    for (int y = 0; y < HEIGHT; y++) {
        for (int x = 0; x < WIDTH; x++)
            grid[y][x] = ' ';
        grid[y][WIDTH] = '\0';
    }

    /* Borders */
    for (int x = 0; x < WIDTH; x++) { grid[0][x] = '-'; grid[HEIGHT - 1][x] = '-'; }
    for (int y = 0; y < HEIGHT; y++) { grid[y][0] = '|'; grid[y][WIDTH - 1] = '|'; }
    grid[0][0] = '+'; grid[0][WIDTH - 1] = '+';
    grid[HEIGHT - 1][0] = '+'; grid[HEIGHT - 1][WIDTH - 1] = '+';

    for (int i = 0; i < g->num_walls; i++) {
        Pos w = g->walls[i];
        if (w.x > 0 && w.x < WIDTH - 1 && w.y > 0 && w.y < HEIGHT - 1)
            grid[w.y][w.x] = '#';
    }
    for (int i = 0; i < g->num_berries; i++) {
        Pos b = g->berries[i];
        if (b.x > 0 && b.x < WIDTH - 1 && b.y > 0 && b.y < HEIGHT - 1)
            grid[b.y][b.x] = 'o';
    }
    for (int i = 0; i < g->num_enemies; i++) {
        Pos e = g->enemies[i].pos;
        if (e.x > 0 && e.x < WIDTH - 1 && e.y > 0 && e.y < HEIGHT - 1)
            grid[e.y][e.x] = 'X';
    }
    if (g->player.x > 0 && g->player.x < WIDTH - 1 && g->player.y > 0 && g->player.y < HEIGHT - 1)
        grid[g->player.y][g->player.x] = '@';

    printf("\033[H\033[2J");
    for (int y = 0; y < HEIGHT; y++)
        printf("%s\r\n", grid[y]);

    printf("\r\n+---------------------------------------+\r\n");
    printf("| Level: %-3d  Score: %-6d  Berries: %d/%d\r\n",
           g->level, g->score, g->berries_needed - g->num_berries, g->berries_needed);
    printf("+---------------------------------------+\r\n");
    printf("\r\nControls: W=Up, S=Down, A=Left, D=Right, Q=Quit\r\n");

    if (g->game_over)
        printf("\r\n*** GAME OVER! You were caught by an alien! ***\r\n");
    if (g->won)
        printf("\r\n*** LEVEL COMPLETE! Press any key for next level ***\r\n");

    fflush(stdout);
}

static void move_player(Game *g, int dx, int dy) {
    int nx = g->player.x + dx, ny = g->player.y + dy;
    if (nx <= 0 || nx >= WIDTH - 1 || ny <= 0 || ny >= HEIGHT - 1) return;
    if (is_wall(g, nx, ny)) return;

    g->player.x = nx;
    g->player.y = ny;

    for (int i = 0; i < g->num_berries; i++) {
        if (g->berries[i].x == nx && g->berries[i].y == ny) {
            g->berries[i] = g->berries[--g->num_berries];
            g->score += 10;
            break;
        }
    }
    if (g->num_berries == 0) g->won = 1;
}

static void move_enemies(Game *g) {
    static const int dxs[] = {0, 1, 0, -1};
    static const int dys[] = {-1, 0, 1, 0};

    for (int i = 0; i < g->num_enemies; i++) {
        Enemy *e = &g->enemies[i];

        if ((rand() % 100) < 30) {
            if (g->player.x > e->pos.x) e->dir = 1;
            else if (g->player.x < e->pos.x) e->dir = 3;
            else if (g->player.y > e->pos.y) e->dir = 2;
            else if (g->player.y < e->pos.y) e->dir = 0;
        }

        int nx = e->pos.x + dxs[e->dir];
        int ny = e->pos.y + dys[e->dir];

        if (nx > 0 && nx < WIDTH - 1 && ny > 0 && ny < HEIGHT - 1 && !is_wall(g, nx, ny)) {
            e->pos.x = nx;
            e->pos.y = ny;
        } else {
            e->dir = rand() % 4;
        }

        if (e->pos.x == g->player.x && e->pos.y == g->player.y)
            g->game_over = 1;
    }
}

static int read_key(void) {
    struct pollfd pfd = {STDIN_FILENO, POLLIN, 0};
    if (poll(&pfd, 1, 0) > 0) {
        char c;
        if (read(STDIN_FILENO, &c, 1) == 1) return c;
    }
    return -1;
}

int main(void) {
    srand(time(NULL));
    raw_mode();

    Game game = {.level = 1};
    init_level(&game);
    draw(&game);

    while (!game.game_over) {
        int key = read_key();

        if (key != -1) {
            if (game.won) {
                game.level++;
                init_level(&game);
                draw(&game);
                continue;
            }
            switch (key) {
                case 'w': case 'W': move_player(&game, 0, -1); break;
                case 's': case 'S': move_player(&game, 0, 1); break;
                case 'a': case 'A': move_player(&game, -1, 0); break;
                case 'd': case 'D': move_player(&game, 1, 0); break;
                case 'q': case 'Q':
                    printf("\r\nThanks for playing XZAP!\r\n");
                    return 0;
            }
            draw(&game);
        }

        usleep(50000); /* 50ms poll interval */

        /* Move enemies every ~300ms (6 ticks) */
        static int tick = 0;
        if (++tick >= 6 && !game.won) {
            tick = 0;
            move_enemies(&game);
            draw(&game);
        }
    }

    draw(&game);
    printf("\r\nFinal Score: %d\r\n", game.score);
    printf("Thanks for playing XZAP!\r\n");
    return 0;
}

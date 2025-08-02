package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var tm = &TextureManager{
	textures: make(map[string]*Texture),
}

type Animated struct {
	CurrentFrame  int
	IsPlaying     bool
	StartTime     time.Time
	FrameDelay    time.Duration
	FrameTextures []*Texture
	Reversing     bool
}

type Player struct {
	Stand     Animated
	Hit       Animated
	Move      Animated
	Pos       rl.Vector2
	DefPos    rl.Vector2
	Speed     float32
	Rotation  float32
	Flip      bool
	Scale     float32
	VelocityY float32
	OnGround  bool
	State     *PlayerState
}

const (
	gravity   = 0.5
	jumpForce = -12
)

var (
	player     Player
	background *Animated
	screenSize rl.Vector2
	music      rl.Music
)

func main() {
	screenSize = rl.NewVector2(1920, 1080)
	rl.InitWindow(int32(screenSize.X), int32(screenSize.Y), "Raylib - Mohamed Sheta")
	rl.ToggleFullscreen()
	rl.SetTargetFPS(60)

	rl.InitAudioDevice()
	defer rl.CloseAudioDevice()

	// Signal handling for safe shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		UnloadAssets()
		rl.CloseAudioDevice()
		rl.CloseWindow()
		os.Exit(0)
	}()

	LoadAssets()
	defer UnloadAssets()
	defer rl.CloseWindow()

	LoadMusic()

	for !rl.WindowShouldClose() {
		rl.UpdateMusicStream(music)
		Update()
		Draw()
	}
}

func LoadMusic() {
	music = rl.LoadMusicStream("assets/music/m.mp3")
	rl.PlayMusicStream(music)
}

func LoadAssets() {
	player = Player{
		Pos:      rl.NewVector2(10, screenSize.Y-120),
		DefPos:   rl.NewVector2(10, screenSize.Y-120),
		Speed:    5,
		Rotation: 0,
		Flip:     false,
		Scale:    0.12,
		State:    NewPlayerState(),
		Hit:      Animated{FrameDelay: 80 * time.Millisecond},
		Stand:    Animated{FrameDelay: 150 * time.Millisecond},
		Move:     Animated{FrameDelay: 50 * time.Millisecond, Reversing: true},
	}

	var baseW int32 = 1024
	var baseH int32 = 1024

	loadFrames := func(paths []string, target *[]*Texture) {
		for _, path := range paths {
			fullPath := "assets/images/" + path
			t := tm.Acquire(fullPath, baseW, baseH)
			*target = append(*target, t)
		}
	}

	loadFrames([]string{"stand1.png", "stand2.png", "stand3.png", "stand4.png"}, &player.Stand.FrameTextures)
	loadFrames([]string{"hit1.png", "hit2.png", "hit3.png", "hit4.png"}, &player.Hit.FrameTextures)
	loadFrames([]string{"mv1.png", "mv2.png", "mv3.png", "mv4.png", "mv4.png", "mv5.png", "mv4.png", "mv6.png"}, &player.Move.FrameTextures)

	if len(player.Stand.FrameTextures) > 0 {
		player.Stand.IsPlaying = true
		player.Stand.StartTime = time.Now()
	}

	background = LoadGIFAsAnimated("assets/images/a.gif", 100*time.Millisecond)
}

func UnloadAssets() {
	// Automatically unload all tracked textures
	tm.ReleaseAll()

	// Handle background separately if it's not managed by texture manager
	for _, frame := range background.FrameTextures {
		rl.UnloadTexture(frame.Texture)
	}
}

// Enhanced LoadSafeTextureFromImage that works with the manager
func LoadSafeTextureFromImage(path string, width int32, height int32) *Texture {
	return tm.Acquire(path, width, height)
}

func Draw() {
	rl.BeginDrawing()
	rl.ClearBackground(rl.Black)

	DrawBackgroundGIF(background)

	DrawPlayer()

	rl.EndDrawing()
}

func DrawPlayer() {
	var anim *Animated
	if player.Hit.IsPlaying && player.Hit.CurrentFrame < len(player.Hit.FrameTextures) {
		anim = &player.Hit
	} else if rl.IsKeyDown(rl.KeyLeft) || rl.IsKeyDown(rl.KeyRight) || rl.IsKeyDown(rl.KeyA) || rl.IsKeyDown(rl.KeyD) {
		anim = &player.Move
	} else {
		anim = &player.Stand
	}

	if anim == nil || len(anim.FrameTextures) == 0 {
		return
	}

	frame := anim.CurrentFrame % len(anim.FrameTextures)
	tex := anim.FrameTextures[frame]
	if !tex.Loaded {
		return
	}

	width := float32(tex.Texture.Width) * player.Scale
	height := float32(tex.Texture.Height) * player.Scale
	src := rl.NewRectangle(0, 0, float32(tex.Texture.Width), float32(tex.Texture.Height))
	if player.Flip {
		src.Width *= -1
		src.X = float32(tex.Texture.Width)
	}
	dst := rl.NewRectangle(player.Pos.X, player.Pos.Y, width, height)
	origin := rl.NewVector2(0, 0)
	rl.DrawTexturePro(tex.Texture, src, dst, origin, player.Rotation, rl.White)
}

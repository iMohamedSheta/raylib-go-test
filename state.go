package main

import (
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type PlayerState struct {
	IsMoving bool
}

func NewPlayerState() *PlayerState {
	return &PlayerState{
		IsMoving: false,
	}
}

func Update() {
	now := time.Now()
	player.State.IsMoving = false
	HandleMovement(now)
	ApplyGravity()
	HandleJump()
	HandleHitAnimation(now)
	HandleStandAnimation(now)
	UpdateBackground(now)
}

func HandleMovement(now time.Time) {
	updateWidth := func() float32 {
		if len(player.Move.FrameTextures) == 0 {
			return 0
		}
		return float32(player.Move.FrameTextures[0].Texture.Width) * player.Scale
	}

	width := updateWidth()

	if rl.IsKeyDown(rl.KeyLeft) || rl.IsKeyDown(rl.KeyA) {
		if player.Pos.X > 0 {
			player.Pos.X -= player.Speed
			player.State.IsMoving = true
		}
		player.Flip = true
	}

	if rl.IsKeyDown(rl.KeyRight) || rl.IsKeyDown(rl.KeyD) {
		if player.Pos.X+width < screenSize.X {
			player.Pos.X += player.Speed
			player.State.IsMoving = true
		}
		player.Flip = false
	}

	updateAnimation(&player.Move, player.State.IsMoving && !player.Hit.IsPlaying, now)
}

func ApplyGravity() {
	player.VelocityY += gravity
	player.Pos.Y += player.VelocityY

	if player.Pos.Y >= player.DefPos.Y {
		player.Pos.Y = player.DefPos.Y
		player.VelocityY = 0
		player.OnGround = true
	} else {
		player.OnGround = false
	}
}

func HandleJump() {
	if (rl.IsKeyPressed(rl.KeySpace) || rl.IsKeyPressed(rl.KeyUp)) && player.OnGround {
		player.VelocityY = jumpForce
		player.OnGround = false
	}
}

func HandleHitAnimation(now time.Time) {
	if rl.IsKeyPressed(rl.KeyF) && !player.Hit.IsPlaying {
		player.Hit.IsPlaying = true
		player.Hit.Reversing = false
		player.Hit.CurrentFrame = 2
		player.Hit.StartTime = now
	}

	if player.Hit.IsPlaying && time.Since(player.Hit.StartTime) > player.Hit.FrameDelay {
		player.Hit.StartTime = now
		if player.Hit.Reversing {
			player.Hit.CurrentFrame--
			if player.Hit.CurrentFrame <= 0 {
				player.Hit.IsPlaying = false
				player.Hit.CurrentFrame = 0
				player.Hit.Reversing = false
			}
		} else {
			player.Hit.CurrentFrame++
			if player.Hit.CurrentFrame >= len(player.Hit.FrameTextures) {
				player.Hit.CurrentFrame = len(player.Hit.FrameTextures) - 1
				player.Hit.Reversing = true
			}
		}
	}
}

func HandleStandAnimation(now time.Time) {
	if !player.State.IsMoving && !player.Hit.IsPlaying {
		updateAnimation(&player.Stand, true, now)
	}
}

func UpdateBackground(now time.Time) {
	updateAnimation(background, true, now)
}

func updateAnimation(anim *Animated, shouldUpdate bool, now time.Time) {
	if shouldUpdate {
		if time.Since(anim.StartTime) > anim.FrameDelay {
			anim.StartTime = now
			anim.CurrentFrame = (anim.CurrentFrame + 1) % len(anim.FrameTextures)
		}
	} else {
		anim.CurrentFrame = 0
		anim.StartTime = now
	}
}

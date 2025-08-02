package main

import (
	"image/gif"
	"os"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func LoadGIFAsAnimated(path string, frameDelay time.Duration) *Animated {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	gifImg, err := gif.DecodeAll(file)
	if err != nil {
		panic(err)
	}

	var textures []*Texture
	for _, frame := range gifImg.Image {
		img := rl.NewImageFromImage(frame)
		tex := rl.LoadTextureFromImage(img)
		rl.UnloadImage(img)

		textures = append(textures, &Texture{
			Texture: tex,
			Loaded:  true,
			refs:    1,
		})
	}

	return &Animated{
		CurrentFrame:  0,
		IsPlaying:     true,
		StartTime:     time.Now(),
		FrameDelay:    frameDelay,
		FrameTextures: textures,
		Reversing:     false,
	}
}

func DrawBackgroundGIF(g *Animated) {
	dst := rl.NewRectangle(0, 0, screenSize.X, screenSize.Y)
	if len(g.FrameTextures) == 0 {
		return
	}

	updateAnimation(g, true, time.Now())

	frame := g.FrameTextures[g.CurrentFrame]
	src := rl.NewRectangle(0, 0, float32(frame.Texture.Width), float32(frame.Texture.Height))
	rl.DrawTexturePro(frame.Texture, src, dst, rl.NewVector2(0, 0), 0, rl.White)
}

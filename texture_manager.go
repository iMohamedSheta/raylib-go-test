package main

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// TextureManager manages loading, tracking, and unloading of textures
type TextureManager struct {
	textures map[string]*Texture
}

// Texture holds a Raylib texture and related metadata
type Texture struct {
	Texture rl.Texture2D
	Loaded  bool
	Err     error
	refs    int // reference count
}

// NewTextureManager creates and returns a new TextureManager
func NewTextureManager() *TextureManager {
	return &TextureManager{
		textures: make(map[string]*Texture),
	}
}

// Acquire loads the texture from the given path if not already loaded,
// increments the reference count, and returns the texture handle.
// If the given width and height > 0, it resizes the image before loading the texture.
func (tm *TextureManager) Acquire(path string, width int32, height int32) *Texture {
	if handle, ok := tm.textures[path]; ok {
		handle.refs++
		return handle
	}

	handle := &Texture{refs: 1}
	tm.textures[path] = handle

	img := rl.LoadImage(path)
	if img.Data == nil {
		handle.Err = fmt.Errorf("failed to load image: %s", path)
		handle.Loaded = false
		return handle
	}

	if width > 0 && height > 0 {
		rl.ImageResize(img, width, height)
	}

	handle.Texture = rl.LoadTextureFromImage(img)
	handle.Loaded = true
	rl.UnloadImage(img)

	return handle
}

// Release decrements the reference count for the texture at the given path.
// If the reference count reaches zero, it unloads the texture and removes it from the manager.
func (tm *TextureManager) Release(path string) {
	handle, ok := tm.textures[path]
	if !ok {
		return
	}

	handle.refs--
	if handle.refs <= 0 {
		if handle.Loaded {
			rl.UnloadTexture(handle.Texture)
		}
		delete(tm.textures, path)
	}
}

// ReleaseAll unloads all loaded textures and clears the texture map.
func (tm *TextureManager) ReleaseAll() {
	for path, handle := range tm.textures {
		if handle.Loaded {
			rl.UnloadTexture(handle.Texture)
		}
		delete(tm.textures, path)
	}
}

package loader

import (
	"fmt"
	"sync"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Asset priority levels for memory management
type AssetPriority int

const (
	PriorityLow AssetPriority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical // Never unload
)

// Asset states for streaming
type AssetState int

const (
	StateUnloaded AssetState = iota
	StateLoading
	StateLoaded
	StateUnloading
)

// Asset Manager manager assets loading and their state
type AssetManager struct {
	mu          sync.RWMutex
	assets      map[string]*Asset
	memoryUsed  int64
	memoryLimit int64

	loadQueue   chan string // Queue for loading assets
	unloadQueue chan string // Queue for unloading assets

	// Reference counting for shared assets
	refCounts map[string]int

	// LRU cache for automatic unloading
	accessOrder []string
	maxCached   int
}

type Asset struct {
	ID       string
	Texture  rl.Texture2D
	State    AssetState
	Priority AssetPriority
	Size     int64
	LastUsed time.Time
	RefCount int

	// Async loading
	LoadFuture chan bool
}

// Global professional asset manager
var AssetManagerGlobal = &AssetManager{
	assets:      make(map[string]*Asset),
	memoryLimit: 512 * 1024 * 1024, // 512MB limit
	loadQueue:   make(chan string, 100),
	unloadQueue: make(chan string, 100),
	refCounts:   make(map[string]int),
	maxCached:   50,
}

// Initialize the asset system (called at game startup)
func (am *AssetManager) Initialize() {
	// Start background workers for async loading/unloading
	go am.loadWorker()
	go am.unloadWorker()
	go am.memoryManager()
}

// Request an asset (main interface used by game systems)
func (am *AssetManager) RequestAsset(assetID string, priority AssetPriority) *Asset {
	am.mu.Lock()
	defer am.mu.Unlock()

	asset, exists := am.assets[assetID]

	if !exists {
		// Create new asset entry
		asset = &Asset{
			ID:         assetID,
			State:      StateUnloaded,
			Priority:   priority,
			LoadFuture: make(chan bool, 1),
		}
		am.assets[assetID] = asset
	}

	// Increment reference count
	am.refCounts[assetID]++
	asset.RefCount++
	asset.LastUsed = time.Now()

	// Update LRU cache
	am.updateAccessOrder(assetID)

	// Start loading if not loaded
	if asset.State == StateUnloaded {
		asset.State = StateLoading
		select {
		case am.loadQueue <- assetID:
		default:
			// Queue full, load synchronously
			am.loadAssetSync(assetID)
		}
	}

	return asset
}

// Release an asset reference
func (am *AssetManager) ReleaseAsset(assetID string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if asset, exists := am.assets[assetID]; exists {
		am.refCounts[assetID]--
		asset.RefCount--

		// If no more references and not critical, mark for potential unloading
		if asset.RefCount <= 0 && asset.Priority != PriorityCritical {
			select {
			case am.unloadQueue <- assetID:
			default:
				// Unload queue full, skip for now
			}
		}
	}
}

// Background worker for loading assets
func (am *AssetManager) loadWorker() {
	for assetID := range am.loadQueue {
		am.loadAssetSync(assetID)
	}
}

// Background worker for unloading assets
func (am *AssetManager) unloadWorker() {
	for assetID := range am.unloadQueue {
		// Wait a bit before unloading (in case asset is requested again)
		time.Sleep(5 * time.Second)

		am.mu.Lock()
		if asset, exists := am.assets[assetID]; exists && asset.RefCount <= 0 {
			am.unloadAssetSync(assetID)
		}
		am.mu.Unlock()
	}
}

// Memory manager that automatically unloads assets when memory is low
func (am *AssetManager) memoryManager() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		am.mu.Lock()

		if am.memoryUsed > am.memoryLimit {
			// Free memory by unloading least recently used assets
			am.freeMemoryLRU()
		}

		am.mu.Unlock()
	}
}

// Load asset synchronously
func (am *AssetManager) loadAssetSync(assetID string) {
	asset := am.assets[assetID]
	if asset == nil || asset.State != StateLoading {
		return
	}

	// Simulate loading based on asset type
	var texture rl.Texture2D
	var size int64

	// Example loading logic
	switch {
	case assetID[:9] == "character":
		img := rl.LoadImage("assets/images/character.png")
		texture = rl.LoadTextureFromImage(img)
		rl.UnloadImage(img)
		size = int64(texture.Width * texture.Height * 4) // Approximate size

	case assetID[:3] == "hit":
		path := fmt.Sprintf("assets/images/%s.png", assetID)
		img := rl.LoadImage(path)
		texture = rl.LoadTextureFromImage(img)
		rl.UnloadImage(img)
		size = int64(texture.Width * texture.Height * 4)

	default:
		// Default loading
		img := rl.LoadImage(fmt.Sprintf("assets/images/%s.png", assetID))
		texture = rl.LoadTextureFromImage(img)
		rl.UnloadImage(img)
		size = int64(texture.Width * texture.Height * 4)
	}

	am.mu.Lock()
	asset.Texture = texture
	asset.Size = size
	asset.State = StateLoaded
	am.memoryUsed += size
	am.mu.Unlock()

	// Notify waiting systems
	select {
	case asset.LoadFuture <- true:
	default:
	}
}

// Unload asset synchronously
func (am *AssetManager) unloadAssetSync(assetID string) {
	asset := am.assets[assetID]
	if asset == nil || asset.State != StateLoaded {
		return
	}

	asset.State = StateUnloading
	rl.UnloadTexture(asset.Texture)
	am.memoryUsed -= asset.Size
	asset.State = StateUnloaded
}

// Update LRU access order
func (am *AssetManager) updateAccessOrder(assetID string) {
	// Remove from current position
	for i, id := range am.accessOrder {
		if id == assetID {
			am.accessOrder = append(am.accessOrder[:i], am.accessOrder[i+1:]...)
			break
		}
	}

	// Add to end (most recently used)
	am.accessOrder = append(am.accessOrder, assetID)

	// Trim if too large
	if len(am.accessOrder) > am.maxCached {
		am.accessOrder = am.accessOrder[1:]
	}
}

// Free memory using LRU strategy
func (am *AssetManager) freeMemoryLRU() {
	targetMemory := am.memoryLimit * 80 / 100 // Free to 80% of limit

	for i := 0; i < len(am.accessOrder) && am.memoryUsed > targetMemory; i++ {
		assetID := am.accessOrder[i]
		if asset, exists := am.assets[assetID]; exists {
			if asset.RefCount <= 0 && asset.Priority != PriorityCritical {
				am.unloadAssetSync(assetID)
			}
		}
	}
}

// High-level game system interface
type GameAssetSystem struct {
	manager     *AssetManager
	ownedAssets []string // Track assets owned by this system
}

func NewGameAssetSystem() *GameAssetSystem {
	return &GameAssetSystem{
		manager:     AssetManagerGlobal,
		ownedAssets: make([]string, 0),
	}
}

// Load assets for a specific game system (e.g., character system, UI system)
func (gas *GameAssetSystem) LoadSystemAssets(assetList []string, priority AssetPriority) {
	for _, assetID := range assetList {
		asset := gas.manager.RequestAsset(assetID, priority)
		gas.ownedAssets = append(gas.ownedAssets, assetID)

		// Wait for loading if high priority
		if priority >= PriorityHigh {
			<-asset.LoadFuture
		}
	}
}

// Unload all assets owned by this system
func (gas *GameAssetSystem) UnloadSystemAssets() {
	for _, assetID := range gas.ownedAssets {
		gas.manager.ReleaseAsset(assetID)
	}
	gas.ownedAssets = gas.ownedAssets[:0]
}

// Get a loaded texture (returns immediately)
func (gas *GameAssetSystem) GetTexture(assetID string) (rl.Texture2D, bool) {
	gas.manager.mu.RLock()
	defer gas.manager.mu.RUnlock()

	if asset, exists := gas.manager.assets[assetID]; exists && asset.State == StateLoaded {
		asset.LastUsed = time.Now()
		return asset.Texture, true
	}

	return rl.Texture2D{}, false
}

// Example usage in your game
func ExampleProfessionalUsage() {
	// Initialize the system
	AssetManagerGlobal.Initialize()

	// Create systems for different parts of your game
	characterSystem := NewGameAssetSystem()
	uiSystem := NewGameAssetSystem()

	// Load assets per system
	characterAssets := []string{"character", "hit1", "hit2", "hit3", "hit4", "move2", "move3"}
	characterSystem.LoadSystemAssets(characterAssets, PriorityHigh)

	uiAssets := []string{"button", "menu_bg", "cursor"}
	uiSystem.LoadSystemAssets(uiAssets, PriorityMedium)

	// Use assets
	if texture, loaded := characterSystem.GetTexture("character"); loaded {
		// Render character
		_ = texture
	}

	// When switching levels or systems
	characterSystem.UnloadSystemAssets() // Auto-releases references
	uiSystem.UnloadSystemAssets()
}

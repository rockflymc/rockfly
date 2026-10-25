package packbuilder

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"strings"
	"sync"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/resource"
)

// DynamicCompiler compiles Bedrock Edition client resource packs (.mcpack) completely in-memory,
// eliminating all disk I/O and temporary directory overhead during runtime pack delivery.
type DynamicCompiler struct {
	mu sync.Mutex

	files         map[string][]byte
	lang          []string
	blockTextures map[string]any
	itemTextures  map[string]any
	assetCount    int
}

// NewDynamicCompiler creates a new in-memory resource pack compiler.
func NewDynamicCompiler() *DynamicCompiler {
	return &DynamicCompiler{
		files:         make(map[string][]byte),
		blockTextures: make(map[string]any),
		itemTextures:  make(map[string]any),
	}
}

// AddFile adds a raw byte buffer at the relative path within the pack.
func (c *DynamicCompiler) AddFile(path string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.files[cleanPath(path)] = bytes.Clone(data)
	c.assetCount++
}

// AddText adds text content at the relative path within the pack.
func (c *DynamicCompiler) AddText(path string, content string) {
	c.AddFile(path, []byte(content))
}

// AddImage encodes an image.Image as a PNG and adds it at the relative path.
func (c *DynamicCompiler) AddImage(path string, img image.Image) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	c.AddFile(path, buf.Bytes())
	return nil
}

// AddLang registers a localization entry in the pack's en_US.lang.
func (c *DynamicCompiler) AddLang(key, translation string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lang = append(c.lang, fmt.Sprintf("%s=%s", key, translation))
}

// CustomBlockData holds metadata, textures, and geometry for custom block packaging.
type CustomBlockData struct {
	Name     string
	Textures map[string]image.Image
	Geometry []byte
}

// AddBlockData registers a custom block using pure data structures.
func (c *DynamicCompiler) AddBlockData(identifier string, b CustomBlockData) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	parts := strings.Split(identifier, ":")
	name := parts[len(parts)-1]

	c.lang = append(c.lang, fmt.Sprintf("tile.%s.name=%s", identifier, b.Name))
	for texName, img := range b.Textures {
		c.blockTextures[texName] = map[string]string{"textures": "textures/blocks/" + texName}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return err
		}
		c.files[fmt.Sprintf("textures/blocks/%s.png", texName)] = buf.Bytes()
	}

	if len(b.Geometry) > 0 {
		c.files[fmt.Sprintf("models/blocks/%s.geo.json", name)] = bytes.Clone(b.Geometry)
	}

	c.assetCount++
	return nil
}

// AddCustomBlock registers a custom block's textures, geometry, and language definitions from a CustomBlockBuildable.
func (c *DynamicCompiler) AddCustomBlock(identifier string, b world.CustomBlockBuildable) error {
	return c.AddBlockData(identifier, CustomBlockData{
		Name:     b.Name(),
		Textures: b.Textures(),
		Geometry: b.Geometry(),
	})
}

// AssetCount returns the total number of custom features added.
func (c *DynamicCompiler) AssetCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.assetCount
}

// Compile builds a complete in-memory .mcpack zip archive, computes its deterministic SHA-256 hash,
// and returns a ready-to-stream *resource.Pack.
func (c *DynamicCompiler) Compile(packName, packDescription string) (*resource.Pack, [32]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. Pack Icon
	if len(packIcon) > 0 {
		c.files["pack_icon.png"] = bytes.Clone(packIcon)
	}

	// 2. Terrain texture atlas
	if len(c.blockTextures) > 0 {
		atlasData, err := json.Marshal(map[string]any{
			"resource_pack_name": "vanilla",
			"texture_name":       "atlas.terrain",
			"padding":            8,
			"num_mip_levels":     4,
			"texture_data":       c.blockTextures,
		})
		if err != nil {
			return nil, [32]byte{}, err
		}
		c.files["textures/terrain_texture.json"] = atlasData
	}

	// 3. Item texture atlas
	if len(c.itemTextures) > 0 {
		atlasData, err := json.Marshal(map[string]any{
			"resource_pack_name": "vanilla",
			"texture_name":       "atlas.items",
			"texture_data":       c.itemTextures,
		})
		if err != nil {
			return nil, [32]byte{}, err
		}
		c.files["textures/item_texture.json"] = atlasData
	}

	// 4. Language file
	if len(c.lang) > 0 {
		c.files["texts/en_US.lang"] = []byte(strings.Join(c.lang, "\n"))
	}

	// 5. Compute deterministic hash over sorted file entries
	contentHasher := sha256.New()
	for p, data := range c.files {
		contentHasher.Write([]byte(p))
		contentHasher.Write(data)
	}
	hashBytes := contentHasher.Sum(nil)
	var contentHash [32]byte
	copy(contentHash[:], hashBytes)

	// Derive deterministic UUIDs from content hash
	headerUUID := uuid.NewHash(sha256.New(), uuid.Nil, append(hashBytes, []byte("header")...), 5)
	moduleUUID := uuid.NewHash(sha256.New(), uuid.Nil, append(hashBytes, []byte("module")...), 5)

	// 6. Generate manifest.json
	manifestData, err := json.Marshal(resource.Manifest{
		FormatVersion: 2,
		Header: resource.Header{
			Name:               packName,
			Description:        packDescription,
			UUID:               headerUUID,
			Version:            [3]int{1, 0, 0},
			MinimumGameVersion: parseVersion(protocol.CurrentVersion),
		},
		Modules: []resource.Module{
			{
				UUID:        moduleUUID.String(),
				Description: packDescription,
				Type:        "resources",
				Version:     [3]int{1, 0, 0},
			},
		},
	})
	if err != nil {
		return nil, [32]byte{}, err
	}
	c.files["manifest.json"] = manifestData

	// 7. Write to in-memory zip archive
	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)

	for p, data := range c.files {
		w, err := zw.Create(p)
		if err != nil {
			return nil, [32]byte{}, err
		}
		if _, err := w.Write(data); err != nil {
			return nil, [32]byte{}, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, [32]byte{}, err
	}

	// 8. Read directly with gophertunnel resource.Read
	pack, err := resource.Read(bytes.NewReader(zipBuf.Bytes()))
	if err != nil {
		return nil, [32]byte{}, err
	}

	return pack, contentHash, nil
}

func cleanPath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	return strings.TrimPrefix(p, "/")
}

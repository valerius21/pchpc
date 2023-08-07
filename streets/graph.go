package streets

import (
	"strconv"

	"github.com/dominikbraun/graph"
	"github.com/rs/zerolog/log"

	"pchpc/utils"
)

// EdgeData is the data stored in an edge
type EdgeData struct {
	MaxSpeed float64
	Length   float64
	Map      *utils.HashMap[string, *Vehicle]
}

// NewGraphFromJSON creates a new graph from a JSON input
func NewGraphFromJSON(jsonBytes []byte) (graph.Graph[int, GVertex], error) {
	json, err := UnmarshalGraphJSON(jsonBytes)
	if err != nil {
		return nil, err
	}
	return NewGraph(json.Graph.Vertices, json.Graph.Edges), nil
}

// NewGraph creates a new graph
func NewGraph(vertices []JVertex, edges []JEdge) graph.Graph[int, GVertex] {
	log.Info().Msg("Creating new graph.")
	vertexHash := func(vertex GVertex) int {
		return vertex.ID
	}
	g := graph.New(vertexHash, graph.Directed())

	for _, vertex := range vertices {
		_ = g.AddVertex(GVertex{
			ID: vertex.OsmID,
			X:  vertex.X,
			Y:  vertex.Y,
		})
	}

	for _, edge := range edges {
		msi, err := strconv.Atoi(edge.MaxSpeed)
		msf := float64(msi)
		if err != nil {
			msf = 50.0 // Default max speed, aka. 'the inchident'
		}

		hMap := utils.NewMap[string, *Vehicle]()
		_ = g.AddEdge(
			edge.From,
			edge.To,
			graph.EdgeData(EdgeData{
				MaxSpeed: msf,
				Length:   edge.Length,
				Map:      &hMap,
			}))
	}

	return g
}

// GetVertices returns a list of vertices in the graph
func GetVertices(g *graph.Graph[int, GVertex]) ([]GVertex, error) {
	gg := *g
	edges, err := gg.Edges()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get edges.")
		return nil, err
	}

	vertices := make(map[int]bool, 0)

	for _, edge := range edges {
		src := edge.Source
		dst := edge.Target

		vertices[src] = false
		vertices[dst] = false
	}

	keys := make([]int, 0, len(vertices))
	for k := range vertices {
		keys = append(keys, k)
	}

	gVertices := make([]GVertex, 0, len(vertices))
	for _, key := range keys {
		vertex, err := gg.Vertex(key)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get vertex.")
			return nil, err
		}
		gVertices = append(gVertices, vertex)
	}

	return gVertices, nil
}

// GetTopRightBottomLeftVertices returns the top right and bottom left vertices of the graph
func GetTopRightBottomLeftVertices(gr *graph.Graph[int, GVertex]) (bot, top Point) {
	// Get all vertices
	vertices, err := GetVertices(gr)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get vertices.")
		return bot, top
	}

	botX := 100.
	botY := 100.
	topX := 0.
	topY := 0.

	for _, vertex := range vertices {

		if vertex.X < botX {
			botX = vertex.X
		}
		if vertex.Y < botY {
			botY = vertex.Y
		}
		if vertex.X > topX {
			topX = vertex.X
		}
		if vertex.Y > topY {
			topY = vertex.Y
		}
	}

	bot = Point{
		X: botX,
		Y: botY,
	}
	top = Point{
		X: topX,
		Y: topY,
	}

	log.Debug().Msgf("Bottom left vertex: %v", bot)
	log.Debug().Msgf("Top right vertex: %v", top)

	return bot, top
}

// Point is a point in 2D space
type Point struct {
	X, Y float64
}

// Rect is a rectangle in 2D space, holding the top right and bottom left points
// and the vertices of the rectangle
type Rect struct {
	TopRight Point
	BotLeft  Point
	Vertices []GVertex
}

// InRect checks if a vertex is in a rectangle
func (r *Rect) InRect(v GVertex) bool {
	for _, vertex := range r.Vertices {
		if vertex.ID == v.ID {
			return true
		}
	}
	return false
}

// DivideGraphsIntoRects divides the graph into n parts. Column-wise division.
func DivideGraphsIntoRects(n int, gr *graph.Graph[int, GVertex]) ([]Rect, error) {
	rootBot, rootTop := GetTopRightBottomLeftVertices(gr)
	// Get all vertices
	vertices, err := GetVertices(gr)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get vertices.")
		return nil, err
	}

	rects := make([]Rect, n)

	xDelta := rootTop.X - rootBot.X

	for i := 0; i < n; i++ {
		botX := rootBot.X + (xDelta/float64(n))*float64(i)
		topX := rootBot.X + (xDelta/float64(n))*float64(i+1)

		rects[i] = Rect{
			TopRight: Point{
				X: topX,
				Y: rootTop.Y,
			},
			BotLeft: Point{
				X: botX,
				Y: rootBot.Y,
			},
			Vertices: make([]GVertex, 0),
		}

		for _, vertex := range vertices {
			isInYInterval := vertex.Y >= rootBot.Y && vertex.Y <= rootTop.Y
			isInXInterval := vertex.X >= botX && vertex.X <= topX

			if isInYInterval && isInXInterval {
				rects[i].Vertices = append(rects[i].Vertices, vertex)
			}
		}
	}

	return rects, nil
}

// GraphFromRect returns a graph from a rectangle
func GraphFromRect(edges []RawEdge[int], rect Rect) graph.Graph[int, GVertex] {
	hashFn := func(v GVertex) int {
		return v.ID
	}

	g := graph.New[int, GVertex](hashFn)

	for _, vertex := range rect.Vertices {
		err := g.AddVertex(vertex)
		if err != nil {
			log.Error().Err(err).Msg("Failed to add vertex.")
			continue
		}
	}

	for _, edge := range edges {
		src := edge.Source
		dst := edge.Target

		srcInRect := false
		dstInRect := false

		for _, vertex := range rect.Vertices {
			if vertex.ID == src {
				srcInRect = true
			}
			if vertex.ID == dst {
				dstInRect = true
			}
		}

		if srcInRect && dstInRect {
			err := g.AddEdge(src, dst)
			if err != nil {
				if err.Error() == "edge already exists" { // annoying
					continue
				}
				log.Error().Err(err).Msg("Failed to add edge.")
				continue
			}
		}
	}

	return g
}

// RawEdge is an edge without magic
type RawEdge[K comparable] struct {
	Source K
	Target K
}

// VertexInGraph checks if a vertex is in a graph
func VertexInGraph(g *graph.Graph[int, GVertex], v GVertex) bool {
	_, err := (*g).Vertex(v.ID)
	if err != nil {
		return false
	}
	return true
}

package streets

// Author: Valerius Mattfeld

import (
	"github.com/aidarkhanov/nanoid"
	"github.com/gammazero/deque"
	"github.com/rs/zerolog/log"
)

const NotInQueue = -1

// Vehicle represents a vehicle in the simulation
type Vehicle struct {
	ID          string
	Speed       float64 // m/s
	Path        Path
	Graph       *Graph
	PathLength  *deque.Deque[float64]
	IsParked    bool
	CurrentEdge *Edge
	// Length?
}

// NewVehicle creates a new vehicle
func NewVehicle(path Path, speed float64, graph Graph) Vehicle {
	v := Vehicle{
		ID:       nanoid.New(),
		Speed:    speed,
		Path:     path,
		Graph:    &graph,
		IsParked: false,
	}

	var q deque.Deque[float64]
	pathLength := v.GetPathLengths()

	for i := 0; i < len(pathLength); i++ {
		if pathLength[i] != 0 {
			q.PushBack(pathLength[i])
		}
	}
	v.PathLength = &q
	return v
}

func (v *Vehicle) Step() {
	// vehicle is at destination
	if v.IsParked {
		return
	}
	v.drive()
}

func (v *Vehicle) drive() {
	v.CurrentEdge = v.GetCurrentEdge()
	if v.CurrentEdge.GetPosition(v) == NotInQueue {
		v.CurrentEdge.PushVehicle(v)
		log.Info().Msgf("Vehicle %v has entered edge %v", v.ID, v.CurrentEdge.ID)
		log.Info().Msgf("Vehicle %v is now at position %v", v.ID, v.CurrentEdge.GetPosition(v))
	}

	//if q.Len() == 0 {
	//	v.IsParked = true
	//	log.Info().Msgf("Vehicle %v has arrived at destination", v.ID)
	//	return
	//}
	q := v.PathLength

	if q.Back() <= v.Speed && q.Len() > 1 {
		backM := q.PopBack()
		bM := q.PopBack()
		q.PushBack(backM + bM)
	} else if q.Back() <= v.Speed && q.Len() == 1 {
		q.PopBack()
		v.CurrentEdge.PopVehicle()
		v.IsParked = true
		log.Info().Msgf("Vehicle %v has arrived at destination", v.ID)
		return
	} else {
		backLength := q.PopBack()
		q.PushBack(backLength - v.Speed)
	}
}

func (v *Vehicle) PrintInfo() {
	if v.CurrentEdge != nil {
		log.Info().Msgf("Vehicle %v: Speed=%v m/s, PathLength=%v m, Edge=%v (N=%d/%d)", v.ID, v.Speed,
			v.PathLength, v.CurrentEdge.ID, v.CurrentEdge.GetPosition(v)+1, v.CurrentEdge.Q.Len())
		return
	}
	log.Info().Msgf("Vehicle %v: Speed=%v m/s, PathLength=%v m, Edge=%v (N=%d)", v.ID, v.Speed,
		v.PathLength, nil, -1)
}

func (v *Vehicle) GetPathLengths() []float64 {
	var lengths []float64
	for i, vertex := range v.Path.Vertices {
		if i == len(v.Path.Vertices)-1 {
			continue
		}
		edge, err := v.Graph.GetCorrespondingEdge(&vertex, &v.Path.Vertices[i+1])
		if err != nil {
			log.Panic().Err(err).Msg("Failed to get corresponding edge")
		}
		lengths = append(lengths, edge.Length)
	}
	return lengths
}

func (v *Vehicle) GetCurrentEdge() *Edge {
	if v.IsParked {
		return nil
	}

	var nonZeroIdx int

	for i := 0; i < v.PathLength.Len(); i++ {
		if v.PathLength.At(i) != 0 {
			nonZeroIdx = i
			break
		}
	}
	for idx, vertex := range v.Path.Vertices {
		if idx == nonZeroIdx {
			if edge, err := v.Graph.GetCorrespondingEdge(&vertex, &v.Path.Vertices[idx+1]); err != nil {
				log.Panic().Err(err).Msg("Failed to get corresponding edge")
			} else {
				return edge
			}
		}
	}
	return nil
}

func (v *Vehicle) IsLeading() bool {
	return v.CurrentEdge.FrontVehicle(v) == nil
}

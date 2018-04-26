package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Node struct {
	Value      int
	Row        int
	Col        int
	IsSource   bool
	Neighbours *[]*Node
}

func (n Node) String() string {
	return fmt.Sprintf("%v", n.Value)
}

func (n *Node) print() string {
	return fmt.Sprintf("{%v, isSource: %v, \tNeighours: %v}", n.Value, n.IsSource, n.Neighbours)
}

type Route struct {
	Steep int
	Nodes *[]*Node
}

var Routes []*Route

func main() {

	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) != 1 {
		fmt.Println("usage: go run main.go <mapfile>")
		return
	}

	graph, err := readMapFile(os.Args[1])
	if err != nil {
		fmt.Printf("could not read mapfile: %v\n", err)
		return
	}

	Routes = make([]*Route, 0)

	//dfs(graph)
	addNeighbours(graph)

	//for _, nodes := range *graph {
	//	for _, node := range nodes {
	//		fmt.Println(node.print())
	//	}
	//}
	dfsAL(graph)

	if len(Routes) == 0 {
		fmt.Println("no routes found!!")
		return
	}

	fmt.Printf("\nfound %d route(s) with length %d and steep %d:\n\n",
		len(Routes), len(*Routes[0].Nodes), Routes[0].Steep)

	for _, Route := range Routes {
		i := len(*Route.Nodes) - 1
		reversed := reverse(Route.Nodes)
		for _, Node := range *reversed {
			arrow := ""
			if i != 0 {
				arrow = "->"
			}
			fmt.Printf("%d %s ", Node.Value, arrow)
			i--
		}
	}
	fmt.Println()
}

func dfs(graph *[][]*Node) {
	for _, nodes := range *graph {
		for _, node := range nodes {
			parents := make(map[*Node]*Node)
			visit(node, graph, parents)
			collectRoutes(parents)
		}
	}
}

func visit(node *Node, graph *[][]*Node, parents map[*Node]*Node) {
	north := node.Row-1 >= 0 && (*graph)[node.Row-1][node.Col].Value < node.Value
	if north {
		parents[(*graph)[node.Row-1][node.Col]] = node
		visit((*graph)[node.Row-1][node.Col], graph, parents)
	}

	east := node.Col+1 < len((*graph)[node.Row]) && (*graph)[node.Row][node.Col+1].Value < node.Value
	if east {
		parents[(*graph)[node.Row][node.Col+1]] = node
		visit((*graph)[node.Row][node.Col+1], graph, parents)
	}

	south := node.Row+1 < len(*graph) && (*graph)[node.Row+1][node.Col].Value < node.Value
	if south {
		parents[(*graph)[node.Row+1][node.Col]] = node
		visit((*graph)[node.Row+1][node.Col], graph, parents)
	}

	west := node.Col-1 >= 0 && (*graph)[node.Row][node.Col-1].Value < node.Value
	if west {
		parents[(*graph)[node.Row][node.Col-1]] = node
		visit((*graph)[node.Row][node.Col-1], graph, parents)
	}
}

func dfsAL(graph *[][]*Node) {

	var wg sync.WaitGroup
	var mux sync.Mutex

	// maximal goroutines at once
	var sem = make(chan int, 100)

	i := 0;

	for _, nodes := range *graph {
		for _, node := range nodes {

			if !node.IsSource {
				continue
			}

			i++

			wg.Add(1)
			go func(node *Node, wg *sync.WaitGroup, mux *sync.Mutex, sem chan int) {
				defer wg.Done()
				sem <- 1

				visited := make(map[*Node]bool)
				parents := make(map[*Node]*Node)

				visitAL(node, parents, visited)

				mux.Lock()
				defer mux.Unlock()
				collectRoutes(parents)

				<- sem
			}(node, &wg, &mux, sem)
		}
	}
	wg.Wait()

	fmt.Printf("goroutines (source-Nodes): %v\n", i)
}

func visitAL(node *Node, parents map[*Node]*Node, visited map[*Node]bool) {

	visited[node] = true

	for _, n := range *node.Neighbours {
		if _, ok := visited[n]; ok {
			continue
		}
		parents[n] = node
		visitAL(n, parents, visited)
	}
}

func collectRoutes(parents map[*Node]*Node) {
	for child := range parents {
		route := &[]*Node{child}
		collectRoutesRec(child, parents, route)
	}
}

func collectRoutesRec(node *Node, parents map[*Node]*Node, route *[]*Node) {

	if parent, ok := parents[node]; ok {
		*route = append(*route, parent)
		collectRoutesRec(parent, parents, route)
		return
	}

	if len(Routes) == 0 {
		Routes = append(Routes, &Route{Steep: steep(route), Nodes: route})
		return
	}

	if len(*route) > len(*Routes[0].Nodes) {
		Routes = nil
		Routes = append(Routes, &Route{Steep: steep(route), Nodes: route})
		return
	}

	if len(*route) == len(*Routes[0].Nodes) {
		newSteep := steep(route)
		currentSteep := Routes[0].Steep

		if newSteep < currentSteep {
			return
		}
		if newSteep > currentSteep {
			Routes = nil
		}
		Routes = append(Routes, &Route{Steep: newSteep, Nodes: route})
	}
}

func steep(route *[]*Node) int {
	if route == nil || len(*route) == 0 {
		return 0
	}
	return (*route)[len(*route)-1].Value - (*route)[0].Value
}

func reverse(route *[]*Node) *[]*Node {
	for i := 0; i < len(*route)/2; i++ {
		j := len(*route) - i - 1
		(*route)[i], (*route)[j] = (*route)[j], (*route)[i]
	}
	return route
}

func readMapFile(mapfile string) (*[][]*Node, error) {
	file, err := os.Open(mapfile)
	if err != nil {
		return nil, fmt.Errorf("could not open mapfile %s: %v", mapfile, err)
	}
	defer file.Close()

	var graph [][]*Node
	row := 0

	s := bufio.NewScanner(file)
	isFirstLine := true
	for s.Scan() {
		columns := strings.Split(s.Text(), " ")

		if isFirstLine {
			ival, err := strconv.Atoi(columns[0])
			if err != nil {
				return nil, fmt.Errorf("could not parse element[%d][0]: %v to int", row, columns[0])
			}
			graph = make([][]*Node, ival)
			isFirstLine = false
			continue
		}

		for col, val := range columns {
			ival, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("could not parse element[%d][%d]: %v to int", row, col, val)
			}
			graph[row] = append(graph[row], &Node{Value: ival, Row: row, Col: col})
		}

		row++
	}

	return &graph, nil
}

func addNeighbours(graph *[][]*Node) {
	for _, nodes := range *graph {
		for _, node := range nodes {

			neighbours := make([]*Node, 0)
			isSource := true

			west := node.Col-1 >= 0
			if west {
				if (*graph)[node.Row][node.Col-1].Value < node.Value {
					neighbours = append(neighbours, (*graph)[node.Row][node.Col-1])
				} else {
					isSource = false
				}
			}

			south := node.Row+1 < len(*graph)
			if south {
				if (*graph)[node.Row+1][node.Col].Value < node.Value {
					neighbours = append(neighbours, (*graph)[node.Row+1][node.Col])
				} else {
					isSource = false
				}
			}

			east := node.Col+1 < len((*graph)[node.Row])
			if east {
				if (*graph)[node.Row][node.Col+1].Value < node.Value {
					neighbours = append(neighbours, (*graph)[node.Row][node.Col+1])
				} else {
					isSource = false
				}
			}

			north := node.Row-1 >= 0
			if north {
				if (*graph)[node.Row-1][node.Col].Value < node.Value {
					neighbours = append(neighbours, (*graph)[node.Row-1][node.Col])
				} else {
					isSource = false
				}
			}

			node.Neighbours = &neighbours
			node.IsSource = isSource
		}
	}
}

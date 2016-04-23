package main

type MkShAndOr struct {
	Pipes []*MkShPipeline
	Ops   []MkShTokenType
}

type MkShPipeline struct {
}

type MkShSimpleCmd struct {
	Words []*ShToken
}

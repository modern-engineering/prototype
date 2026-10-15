#Text: string | bytes

#Module: {
	path:    #Text
	version: #Text
}

#Symbol: {
	package: #Text
	name:    #Text
}

#Node: {
	name:       #Text
	definition: #Symbol
}

#Reference: {
	node:   #Text
	output: #Text
}

#Value: {
	kind: "literal"
	text: #Text
} | {
	kind:      "reference"
	reference: #Reference
}

#Binding: {
	node:  #Text
	input: #Text
	value: #Value
}

#Definition: {
	modules:  *[] | [...#Module]
	nodes:    *[] | [...#Node]
	bindings: *[] | [...#Binding]
}

#Artifact: {
	version:    "0.1"
	id:         *"" | #Text
	revision:   *"" | #Text
	definition: #Definition
}

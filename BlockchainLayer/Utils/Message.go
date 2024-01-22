package Utils

// <REQUEST,o,t,c>
type Request struct {
	Message
	Timestamp int64
	//相当于clientID
	ClientAddr string
}

// <<PRE-PREPARE,v,n,d>,m>
type PrePrepare struct {
	RequestMessage Request
	Digest         string
	SequenceID     int
	Sign           []byte
}

// <PREPARE,v,n,d,i>
type Prepare struct {
	RequestMessage Request
	Digest         string
	SequenceID     int
	NodeID         string
	Sign           []byte
}

// <COMMIT,v,n,D(m),i>
type Commit struct {
	RequestMessage Request
	Digest         string
	SequenceID     int
	NodeID         string
	Sign           []byte
}

type Message struct {
	Content string
	ABlock  Block
	ID      int
}

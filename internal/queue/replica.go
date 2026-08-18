package queue

type ReplicaRole string

const (
	ReplicaLeader   ReplicaRole = "leader"
	ReplicaFollower ReplicaRole = "follower"
)

type Replica struct {
	NodeID string
	Role   ReplicaRole
}

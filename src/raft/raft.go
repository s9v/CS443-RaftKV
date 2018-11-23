package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import "sync"
import "labrpc"
import "time"
import "math/rand"

// import "bytes"
// import "labgob"

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

//
// A Go object implementing a single Raft peer.
//
const smallTimeout = 10 * time.Millisecond

type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	me        int                 // this peer's index into peers[]
        // (not Lab1)
	persister *Persister          // Object to hold this peer's persisted state
        applyCh   chan ApplyMsg

	// Your data here (Lab1, Lab2, Challenge1).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

        // Persisten state on all servers
        currentTerm     int
        votedFor        int
        log             []LogEntry

        // Volatile state on all servers (not Lab1)
        commitIndex     int
        lastApplied     int

        // Volatile state on leaders (not Lab1)
        nextIndex       []int
        matchIndex      []int

        // More...
        role                    ServerRole
        votes                   int
        lastElectionTimer       int64
        electionTimeout         int64
        lastHeartbeatTimer      int64
        heartbeatTimeout        int64
}

type LogEntry struct {
    Command     interface{}
    Term        int
}

type ServerRole int
const (
    Leader = iota
    Candidate
    Follower
)

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool

	// Your code here (Lab1).

        rf.mu.Lock()
        defer rf.mu.Unlock()

        term = rf.currentTerm
        isleader = (rf.role == Leader)

	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (Challenge1).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (Challenge1).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

//
// get last log term
//
func (rf *Raft) getLastLogTerm() int {
    if len(rf.log) == 0 {
        return -1
    }
    return rf.log[ len(rf.log)-1 ].Term
}

//
// get last log index
//
func (rf *Raft) getLastLogIndex() int {
    return len(rf.log) - 1
}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (Lab1, Lab2).

        Term            int
        CandidateId     int
        LastLogIndex    int
        LastLogTerm     int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (Lab1).

        Term            int
        VoteGranted    bool
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
        rf.mu.Lock()
        defer rf.mu.Unlock()

        // DPrintf("RequestVote !!! [for %d, by %d]", rf.me, args.CandidateId)

        reply.Term = rf.currentTerm

        if args.LastLogTerm < rf.getLastLogTerm() ||
            (args.LastLogTerm == rf.getLastLogTerm() &&
             args.LastLogIndex < rf.getLastLogIndex()) {
            // not at least as up-to-date
            reply.VoteGranted = false
        } else if args.Term > rf.currentTerm {
            reply.VoteGranted = true
            rf.votedFor = args.CandidateId
            rf.resetElectionTimer()
            go rf.becomeFollower(args.Term)
        } else if args.Term == rf.currentTerm && rf.votedFor == args.CandidateId {
            reply.VoteGranted = true
        } else {
            reply.VoteGranted = false
        }
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply, term int) {
	for term == rf.currentTerm &&
            !rf.peers[server].Call("Raft.RequestVote", args, reply) { }

        if term == rf.currentTerm {
            if reply.VoteGranted {
                rf.votes++
            }

            if reply.Term > rf.currentTerm {
                rf.becomeFollower(reply.Term)
            }
        }
}

//
// example AppendEntries RPC arguments structure.
// field names must start with capital letters!
//
type AppendEntriesArgs struct {
	// Your data here (Lab1, Lab2).

        Term            int
        LeaderId        int

        PrevLogIndex    int
        PrevLogTerm     int
        Entries         []LogEntry
        LeaderCommit    int
}

//
// example AppendEntries RPC reply structure.
// field names must start with capital letters!
//
type AppendEntriesReply struct {
	// Your data here (Lab1).

        Term            int
        Success         bool
}

//
// example AppendEntries RPC handler.
//
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
        rf.mu.Lock()
        defer rf.mu.Unlock()

        if args.Term < rf.currentTerm {
            reply.Term = rf.currentTerm
            reply.Success = false
            return
        }

        if len(args.Entries) > 0 {

            if args.PrevLogIndex >= len(rf.log) ||
                (args.PrevLogIndex >= 0 && rf.log[args.PrevLogIndex].Term != args.PrevLogTerm) {
                reply.Term = rf.currentTerm
                reply.Success = false
                return
            }

            for i := 0; i < len(args.Entries); i++ {
                j := args.PrevLogIndex + 1 + i;

                if len(rf.log) <= j {
                    rf.log = append(rf.log, args.Entries[i])
                } else if rf.log[j].Term != args.Entries[i].Term {
                    rf.log[j] = args.Entries[i]
                }
            }
        }

        if rf.commitIndex < args.LeaderCommit {
            rf.commitIndex = args.LeaderCommit
            rf.applyCommitted()
        }

        reply.Term = rf.currentTerm
        reply.Success = true

        rf.resetElectionTimer()
        if rf.role != Follower {
            go rf.becomeFollower(args.Term)
        }
}

//
// send AppendEntries RPC to a server
//
func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply, heartbeat bool, term int, role ServerRole, loglen int) {
        // keep retrying
	for  {
            if term != rf.currentTerm || role != rf.role || (!heartbeat && loglen != len(rf.log)) { return }

            args.Term = rf.currentTerm
            args.LeaderId = rf.me

            if heartbeat {
                args.Entries = []LogEntry{}
            } else {
                args.PrevLogIndex = rf.nextIndex[server] - 1
                if args.PrevLogIndex >= 0 && args.PrevLogIndex < len(rf.log) {
                    args.PrevLogTerm = rf.log[args.PrevLogIndex].Term
                }
                slice := rf.log[rf.nextIndex[server]:]
                args.Entries = make([]LogEntry, len(slice))
                copy(args.Entries, slice)
            }
            args.LeaderCommit = rf.commitIndex

            // reply received
            if rf.peers[server].Call("Raft.AppendEntries", args, reply) {
                if reply.Success {
                    if !heartbeat {
                        DPrintf("! ! ! [appendEntries success] to:%d  len(Entries):%d", server, len(args.Entries))
                        rf.nextIndex[server] = len(rf.log)
                        rf.matchIndex[server] = len(rf.log)-1
                        rf.updateCommitIndex()
                    }
                    break
                } else {
                    if reply.Term > rf.currentTerm {
                        // higher term
                        DPrintf("[DAMN.] follower has higher term. -- signed %d", rf.me)
                        go rf.becomeFollower(reply.Term)
                        break
                    } else {
                        // doesnt have that PrevLogIndex
                        rf.nextIndex[server]--
                    }
                }
            }
        }
}

func (rf *Raft) updateCommitIndex() {
    DPrintf("[      Update ~~~~~ Commit ~~~~~~~~ Index      ]")
    for i := 0; i < len(rf.peers); i++ {
        DPrintf("                                     matchIndex[%d]:%d", i, rf.matchIndex[i])
    }

    N := rf.commitIndex
    for i := rf.commitIndex + 1; i < len(rf.log); i++ {
        if rf.log[i].Term == rf.currentTerm {
            ok := 0
            for j := 0; j < len(rf.peers); j++ {
                if rf.me == j || rf.matchIndex[j] >= i {
                    ok++
                }
            }
            if ok > len(rf.peers)/2 {
                N = i
            }
        }
    }

    // DPrintf("/* commitIndex: %d", rf.commitIndex)
    if N > rf.commitIndex {
        // DPrintf("Update CommitIndex FORCES")
        rf.commitIndex = N
    }

    rf.applyCommitted()
}

func (rf *Raft) applyCommitted() {
    for i := rf.lastApplied + 1; i <= rf.commitIndex && i < len(rf.log); i++ {
        rf.applyCh <- ApplyMsg{true, rf.log[i].Command, i+1}
        rf.lastApplied = i
        // DPrintf("[diss to heaven] me:%d   index:%d   cmd:%d", rf.me, i, rf.log[i].Command);
    }
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
        // TODO Do I need this?
        DPrintf("! ! ! [Start] me:%d  term:%d  cmd:%d", rf.me, rf.currentTerm, command)
        rf.mu.Lock()
        defer rf.mu.Unlock()

	index := -1
	term := rf.currentTerm
	isLeader := rf.role == Leader

	// append to log
        if (rf.role == Leader) {
            DPrintf("! ! ! [StartLeader] me:%d  term:%d  cmd:%d", rf.me, rf.currentTerm, command)
            rf.log = append(rf.log, LogEntry{command, rf.currentTerm})
            index = len(rf.log)

            // broadcast AppendEntries
            for i := 0; i < len(rf.peers); i++ {
                if i == rf.me {
                    continue
                }

                args := AppendEntriesArgs{}
                reply := AppendEntriesReply{}
                go rf.sendAppendEntries(i, &args, &reply, false, rf.currentTerm, rf.role, len(rf.log))
            }
        }

	return index, term, isLeader
}

//
// the tester calls Kill() when a Raft instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (rf *Raft) Kill() {
	// Your code here, if desired.
}

//
// become follower
//

func (rf *Raft) becomeFollower(newTerm int) {
    rf.mu.Lock()
    defer rf.mu.Unlock()

    if rf.currentTerm > newTerm {
        return
    }

    rf.role = Follower
    rf.currentTerm = newTerm
    rf.votes = 0

    go rf.detectElectionTimeout(rf.currentTerm, rf.role)
}

//
// become candidate
//

func (rf *Raft) becomeCandidate() {
    DPrintf("[Became Candidate] me:%d", rf.me)
    rf.mu.Lock()
    defer rf.mu.Unlock()

    rf.role = Candidate
    rf.currentTerm++
    rf.votedFor = rf.me
    rf.votes = 1
    rf.resetElectionTimer()

    go rf.detectElectionTimeout(rf.currentTerm, rf.role)
    go rf.detectMajorityVote(rf.currentTerm, rf.role)

    for i := 0; i < len(rf.peers); i++ {
        if i == rf.me {
            continue
        }

        args := RequestVoteArgs{rf.currentTerm, rf.me, rf.getLastLogIndex(), rf.getLastLogTerm()}
        reply := RequestVoteReply{}
        go rf.sendRequestVote(i, &args, &reply, rf.currentTerm)
    }
}

func (rf *Raft) resetElectionTimer() {
    rf.lastElectionTimer = time.Now().UnixNano()
}

func (rf *Raft) detectMajorityVote(term int, role ServerRole) {
    for {
        if term != rf.currentTerm || role != rf.role {
            break
        }

        if rf.votes > len(rf.peers)/2 {
            DPrintf("Got Majority Vote! (%d, term:%d)", rf.me, rf.currentTerm)
            go rf.becomeLeader(rf.currentTerm)
            break
        }

        time.Sleep(smallTimeout)
    }
}

// for both candidate and follower
func (rf *Raft) detectElectionTimeout(term int, role ServerRole) {
    for {
        time.Sleep(smallTimeout)

        if term != rf.currentTerm || role != rf.role {
            break
        }

        if rf.electionTimedOut() {
            // DPrintf("electionTimeout !!! [%d.%d]", rf.me, rf.currentTerm)
            go rf.becomeCandidate()
            break
        }
    }
}

func (rf *Raft) electionTimedOut() bool {
    return time.Now().UnixNano() - rf.lastElectionTimer >= rf.electionTimeout
}

//
// become leader
// term - to make sure only {candidate @ term} becomes {leader @ term}
//
func (rf *Raft) becomeLeader(term int) {
    rf.mu.Lock()
    defer rf.mu.Unlock()

    if term < rf.currentTerm {
        return
    }

    rf.role = Leader

    // reset nextIndex & matchIndex
    for i := 0; i < len(rf.nextIndex); i++ {
        rf.nextIndex[i] = len(rf.log)
        rf.matchIndex[i] = -1
    }

    rf.lastHeartbeatTimer = 0
    go rf.detectHeartbeatTimeout(rf.currentTerm, rf.role)
}

func (rf *Raft) detectHeartbeatTimeout(term int, role ServerRole) {
    for {
        if term != rf.currentTerm || role != rf.role {
            break
        }

        if rf.heartbeatTimedOut() {
            rf.resetHeartbeatTimer()

            // send heartbeats to followers
            for i := 0; i < len(rf.peers); i++ {
                if i == rf.me {
                    continue
                }

                args := AppendEntriesArgs{}
                reply := AppendEntriesReply{}
                go rf.sendAppendEntries(i, &args, &reply, true, rf.currentTerm, rf.role, len(rf.log))
            }
        }

        time.Sleep(smallTimeout)
    }
}

func (rf *Raft) heartbeatTimedOut() bool {
    return time.Now().UnixNano() - rf.lastHeartbeatTimer >= rf.heartbeatTimeout
}

func (rf *Raft) resetHeartbeatTimer() {
    rf.lastHeartbeatTimer = time.Now().UnixNano()
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
        rf.applyCh = applyCh

	// Your initialization code here (Lab1, Lab2, Challenge1).
        rf.log = []LogEntry{}

        rf.electionTimeout = randomElectionTimeout()
        rf.heartbeatTimeout = randomHeartbeatTimeout()
        rf.votedFor = -1
        rf.currentTerm = 0

        rf.lastApplied = -1
        rf.commitIndex = -1
        rf.nextIndex = make([]int, len(peers))
        rf.matchIndex = make([]int, len(peers))

        rf.resetElectionTimer()
        rf.becomeFollower(1)

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

        // DPrintf("peeeeeeeeeeers: %d", len(rf.peers))
        // DPrintf("server index: %d", rf.me)
        // DPrintf("smallTimeout: %d", smallTimeout)
        // DPrintf("electionTimeout: %d", rf.electionTimeout)
        // DPrintf("heartbeatTimeout: %d", rf.heartbeatTimeout)

	return rf
}

func randomHeartbeatTimeout() int64 {
    heartbeatTimeout := int64( (rand.Intn(200) + 100) * int(time.Millisecond) )
    return heartbeatTimeout
}

func randomElectionTimeout() int64 {
    electionTimeout := int64( (rand.Intn(200) + 700) * int(time.Millisecond) )
    return electionTimeout
}

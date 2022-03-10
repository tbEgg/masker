package account

import (
	"container/heap"
	"time"

	"masker/cryption"
)

const (
	updateIntervalSec = 10
	cacheDurationSec  = 120
)

type User struct {
	Id *ID `json:"id"`
}

type UserSet interface {
	AddUser(User) error
	GetUser([]byte) (*ID, int64, bool)
}

type TimedUserSet struct {
	validUserIdList   []*ID
	userHashMap       map[string]indexTimePair
	userHashEntryHeap *hashEntryHeap
}

type indexTimePair struct {
	userIndex int
	timeSec   int64
}

func NewTimedUserSet(userList ...User) (UserSet, error) {
	userSet := new(TimedUserSet)
	userSet.userHashMap = make(map[string]indexTimePair)
	userSet.userHashEntryHeap = newHashEntryHeap(100)

	userSet.validUserIdList = make([]*ID, 0, len(userList))
	for _, user := range userList {
		userSet.validUserIdList = append(userSet.validUserIdList, user.Id)
	}

	go userSet.updateUserHash(time.Tick(updateIntervalSec * time.Second))
	return userSet, nil
}

/**
 * every tick cycle update userSet.userHashMap
 * that is, add new user hash about the just experienced time period and delete old user hash that lose timeliness
 * a user hash will lose timeliness if the time interval between current time and timeSec is larger than cacheDurationSec
 *
 * key of userSet.userHashMap: HMAC(user.Id, timeSec)
 *
 */
func (userSet *TimedUserSet) updateUserHash(tick <-chan time.Time) {
	now := time.Now()

	// timeSecWillBeHashed is the next time sec that will be used to generate user hash
	// time before "timeSecWillBeHashed" has been used
	timeSecWillBeHashed := now.Unix() - cacheDurationSec

	// user hash that associated with timeSecWillBeRemoved is the next hash that will be discarded
	// time before "timeSecWillBeRemoved" has lost timeliness
	timeSecWillBeRemoved := timeSecWillBeHashed

	for {
		// problem:
		// first tick does not come immediately, userHashMap is empty during this time
		// so, request arrives too early will be denied because listener can not find the corresponding user hash
		now = <-tick
		curTimeSec := now.Unix()

		// time sec before "timeSecLoseTimeliness" are all too old to lose timeliness
		// user hash associate with them will be discarded
		timeSecLoseTimeliness := curTimeSec - cacheDurationSec
		for userSet.userHashEntryHeap.Len() > 0 {
			entry := (*userSet.userHashEntryHeap)[0]
			timeSecWillBeRemoved = entry.timeSec
			if timeSecWillBeRemoved >= timeSecLoseTimeliness {
				break
			}

			delete(userSet.userHashMap, entry.userHash)
			heap.Pop(userSet.userHashEntryHeap)
		}

		for userIndex := 0; userIndex < len(userSet.validUserIdList); userIndex++ {
			userSet.generateNewUserHash(timeSecWillBeHashed, curTimeSec, userIndex)
		}
	}
}

func (userSet *TimedUserSet) generateNewUserHash(timeSecWillBeHashed, curTimeSec int64, userIndex int) {
	userIdBytes := userSet.validUserIdList[userIndex].Bytes
	for ; timeSecWillBeHashed < curTimeSec+cacheDurationSec; timeSecWillBeHashed++ {
		userHash := string(cryption.TimeHMACHash(userIdBytes, timeSecWillBeHashed))
		userSet.userHashMap[userHash] = indexTimePair{userIndex, timeSecWillBeHashed}
		heap.Push(userSet.userHashEntryHeap, &hashEntry{userHash, timeSecWillBeHashed})
	}
}

func (userSet *TimedUserSet) AddUser(user User) error {
	userIndex := len(userSet.validUserIdList)
	userSet.validUserIdList = append(userSet.validUserIdList, user.Id)

	curTimeSec := time.Now().Unix()
	go userSet.generateNewUserHash(curTimeSec-cacheDurationSec, curTimeSec, userIndex)
	return nil
}

func (userSet *TimedUserSet) GetUser(userHash []byte) (*ID, int64, bool) {
	if pair, ok := userSet.userHashMap[string(userHash)]; ok {
		return userSet.validUserIdList[pair.userIndex], pair.timeSec, true
	} else {
		return nil, 0, false
	}
}

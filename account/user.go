package account

import (
	"time"

	"../cryption"
)

const (
	updateIntervalSec = 10
	cacheDurationSec  = 120
	timeOutSec		  = 5e8
)

type User struct {
	Id	*ID	`json:"id"`
}

type UserSet interface {
	AddUser(User) error
	GetUser([]byte) (*ID, int64, bool)
}

type TimedUserSet struct {
	validUserIdList	[]*ID
	userHashMap		map[string]indexTimePair
}

type indexTimePair struct {
	userIndex	int
	timeSec 	int64
}

type hashEntry struct {
	userHash	string
	timeSec		int64
}

func NewTimedUserSet(userList ...User) (*TimedUserSet, error) {
	userSet := new(TimedUserSet)
	userSet.userHashMap = make(map[string]indexTimePair)
	userSet.validUserIdList = make([]*ID, 0, len(userList))
	
	for _, user := range userList {
		if err := userSet.AddUser(user); err != nil {
			return nil, err
		}
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
	// time before "timeSecWillBeHashed" are already have been used
	timeSecWillBeHashed := now.Unix() - cacheDurationSec

	// user hash that associated with timeSecWillBeRemoved is the next hash that will be discarded
	// time before "timeSecWillBeRemoved" are already losing timeliness
	timeSecWillBeRemoved := timeSecWillBeHashed

	userHashMapEntryChannel := make(chan hashEntry, len(userSet.validUserIdList) * 3 * cacheDurationSec)

	for {
		// problem:
		// first tick does not come immediately, userHashMap is empty during this time
		// so, request arrives too early will be denied because listener can not find the corresponding user
		now = <- tick
		curTimeSec := now.Unix()

		// time sec before "timeSecLoseTimeliness" are all too old to lose timeliness
		// user hash associate with them will be discarded
		timeSecLoseTimeliness := curTimeSec - cacheDurationSec
		for timeSecWillBeRemoved < timeSecLoseTimeliness {
			select {
			case entry := <-userHashMapEntryChannel:
				timeSecWillBeRemoved = entry.timeSec
				delete(userSet.userHashMap, entry.userHash)
				continue	// skip the break statement which is out of the select statement
			case <-time.After(timeOutSec):
				// time out means that userHashMapEntryChannel is empty
				// break to prevent blocking
				break
			}
			break
		}
		
		for timeSecWillBeHashed < curTimeSec + cacheDurationSec {
			for userIndex, userId := range userSet.validUserIdList {
				userHash := string(cryption.TimeHMACHash(userId.Bytes, timeSecWillBeHashed))
				userSet.userHashMap[userHash] = indexTimePair{userIndex, timeSecWillBeHashed}
				userHashMapEntryChannel <- hashEntry{userHash, timeSecWillBeHashed}
			}
			timeSecWillBeHashed++
		}
	}
}

func (userSet *TimedUserSet) AddUser(user User) error {
	userSet.validUserIdList = append(userSet.validUserIdList, user.Id)
	return nil
}

func (userSet *TimedUserSet) GetUser(userHash []byte) (*ID, int64, bool) {
	if pair, ok := userSet.userHashMap[string(userHash)]; ok {
		return userSet.validUserIdList[pair.userIndex], pair.timeSec, true
	} else {
		return nil, 0, false
	}
}
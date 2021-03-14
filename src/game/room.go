package game

import (
	"fmt"
	"time"

	"godori.com/getty"
	toClient "godori.com/packet/toClient"
)

type Room struct {
	Index          int
	RoomType       int
	Max            int
	Mode           *GameMode
	NextEventIndex int
	Places         map[int]*Place
	Events         map[int]*Event
	Users          map[*getty.Client]*User
	Run            bool
	Lock           bool
}

var nextRoomIndex int = 0
var Rooms map[int]*Room = make(map[int]*Room)

func NewRoom(rType int) *Room {
	nextRoomIndex++
	room := &Room{
		Index:    nextRoomIndex,
		RoomType: rType,
		Max:      30,
		Places:   make(map[int]*Place),
		Events:   make(map[int]*Event),
		Users:    make(map[*getty.Client]*User),
		Run:      true,
	}
	Rooms[nextRoomIndex] = room
	room.Mode = NewMode(room)
	go room.Update()
	return room
}

func AvailableRoom(rType int) *Room {
	for index := range Rooms {
		if r, ok := Rooms[index]; ok && r.RoomType == rType && r.CheckJoin() {
			return r
		}
	}
	fmt.Println("새로운 방 만들기 시도 ")
	return NewRoom(rType)
}

func (r *Room) Remove() {
	r.Run = false
	r.Places = make(map[int]*Place)
	r.Events = make(map[int]*Event)
	r.Users = make(map[*getty.Client]*User)
	delete(Rooms, r.Index)
}

func (r *Room) AddEvent(e *Event) {
	e.Room = r
	r.Events[e.EventData.Id] = e
	r.GetPlace(e.Place).AddEvent(e)
}

func (r *Room) RemoveEvent(e *Event) {
	delete(r.Events, e.EventData.Id)
	r.GetPlace(e.Place).RemoveEvent(e)
	e.Room = nil
}

func (r *Room) AddUser(u *User) {
	u.Room = r
	r.Users[u.Client] = u
	r.GetPlace(u.Place).AddUser(u)
}

func (r *Room) RemoveUser(u *User) {
	delete(r.Users, u.Client)
	r.GetPlace(u.Place).RemoveUser(u)
	u.Room = nil
}

func (r *Room) GetPlace(place int) *Place {
	p, ok := r.Places[place]
	if !ok {
		r.Places[place] = NewPlace(place, r)
		p = r.Places[place]
	}
	return p
}

func (r *Room) Publish(d []byte) {
	for _, u := range r.Users {
		u.Send(d)
	}
}

func (r *Room) PublishMap(place int, d []byte) {
	for _, u := range r.GetPlace(place).Users {
		u.Send(d)
	}
}

func (r *Room) Broadcast(self *User, d []byte) {
	for _, u := range r.Users {
		if u == self {
			continue
		}
		u.Send(d)
	}
}

func (r *Room) BroadcastMap(self *User, d []byte) {
	for _, u := range r.GetPlace(self.Place).Users {
		if u == self {
			continue
		}
		u.Send(d)
	}
}

func (r *Room) SameMapUsers(place int) map[*getty.Client]*User {
	return r.GetPlace(place).Users
}

func (r *Room) Passable(place int, x int, y int, dir int, collider bool) bool {
	if collider {
		for _, e := range r.GetPlace(place).Events {
			if e.EventData.Collider && e.X == x && e.Y == y {
				return false
			}
		}
	}
	return GameMaps[place].Passable(x, y, dir)
}

func (r *Room) Portal(u *User) {
	if p, ok := GameMaps[u.Place].GetPortal(u.X, u.Y); ok {
		r.Teleport(u, p.NextPlace, p.NextX, p.NextY, p.NextDirX, p.NextDirY)
		if p.Sound != "" {
			r.PublishMap(u.Place, toClient.PlaySound(p.Sound))
		}
	}
}

func (r *Room) Teleport(u *User, place int, x int, y int, dirX int, dirY int) {
	r.GetPlace(u.Place).RemoveUser(u)
	u.Portal(place, x, y, dirX, dirY)
	r.GetPlace(place).AddUser(u)
	r.Draw(u)
}

func (r *Room) Hit(self *User) {
	for _, u := range r.GetPlace(self.Place).Users {
		if !(self.X == u.X && self.Y == u.Y || self.X+self.DirX == u.X && self.Y-self.DirY == u.Y) {
			continue
		}
		if r.Mode.Hit(self, u) {
			break
		}
	}
	for _, e := range r.GetPlace(self.Place).Events {
		if !(self.X == e.X && self.Y == e.Y || self.X+self.DirX == e.X && self.Y-self.DirY == e.Y) {
			continue
		}
		if e.Do(r, self) {
			break
		}
	}
}

func (r *Room) UseItem(u *User) {
	r.Mode.UseItem(u)
}

func (r *Room) CheckJoin() bool {
	return len(r.Users) < r.Max && !r.Lock
}

func (r *Room) Draw(u *User) {
	r.Mode.DrawEvents(u)
	r.Mode.DrawUsers(u)
}

func (r *Room) Join(u *User) {
	r.AddUser(u)
	r.Mode.Join(u)
	r.Publish(toClient.UpdateRoomUserCount(len(r.Users)))
}

func (r *Room) Leave(u *User) {
	r.Mode.Leave(u)
	r.RemoveUser(u)
	r.PublishMap(u.Place, toClient.RemoveGameObject(u.Model, u.Index))
	r.Publish(toClient.UpdateRoomUserCount(len(r.Users)))
	if len(r.Users) <= 0 {
		r.Remove()
	}
}

func (r *Room) Update() {
	for r.Run {
		for _, p := range r.Places {
			p.Update()
		}
		r.Mode.Update()
		time.Sleep(100 * time.Millisecond)
	}
}

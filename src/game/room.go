package game

import (
	"time"

	"godori.com/getty"
)

type Room struct {
	Index          int
	RoomType       int
	Max            int
	Mode           *GameMode
	NextEventIndex int
	Users          map[*getty.Client]*User
	Places         map[int]*Place
	Run            bool
	Lock           bool
}

var nextRoomIndex int
var Rooms map[int]*Room = make(map[int]*Room)

func NewRoom(rType int) *Room {
	nextRoomIndex++
	room := &Room{
		Index:    nextRoomIndex,
		RoomType: rType,
		Max:      30,
		Users:    make(map[*getty.Client]*User),
		Places:   make(map[int]*Place),
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
	return NewRoom(rType)
}

func (r *Room) Remove() {
	r.Run = false
	delete(Rooms, r.Index)
}

func (r *Room) AddEvent() {
	// TODO : add event
}

func (r *Room) RemoveEvent() {
	// TODO : remove event
}

func (r *Room) AddUser(u *User) {
	u.room = r.Index
	r.Users[u.client] = u
	r.GetPlace(u.place).AddUser(u)
}

func (r *Room) RemoveUser(u *User) {
	delete(r.Users, u.client)
	r.GetPlace(u.place).RemoveUser(u)
	u.room = 0
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
	for _, u := range r.GetPlace(self.place).Users {
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
		// TODO : event
		//r.GetPlace(place).Events
	}
	return GameMaps[place].Passable(x, y, dir)
}

func (r *Room) Portal(u *User) {
	if p, ok := GameMaps[u.place].GetPortal(u.character.x, u.character.y); ok {
		r.Teleport(u, p.NextPlace, p.NextX, p.NextY, p.NextDirX, p.NextDirY)
		if p.Sound != "" {
			// TODO : 사운드 재생
			//r.PublishMap(u.place, )
		}
	}
}

func (r *Room) Teleport(u *User, place int, x int, y int, dirX int, dirY int) {
	r.GetPlace(u.place).RemoveUser(u)
	u.Portal(place, x, y, dirX, dirY)
	r.GetPlace(place).AddUser(u)
	//r.Draw(u)
}

func (r *Room) Hit(u *User) {
	// TODO
	r.Mode.Hit(u, u)
}

func (r *Room) UseItem(u *User) {
	r.Mode.UseItem(u)
}

func (r *Room) CheckJoin() bool {
	return len(r.Users) < r.Max && !r.Lock
}

func (r *Room) Draw(u *User) {
	// TODO : draw
	r.Mode.DrawUsers(u)
}

func (r *Room) Join(u *User) {
	// TODO : join
	r.AddUser(u)
	r.Mode.Join(u)
}

func (r *Room) Leave(u *User) {
	// TODO : leave
	r.Mode.Leave(u)
	r.RemoveUser(u)
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

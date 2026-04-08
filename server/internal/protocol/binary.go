package protocol

import (
	"encoding/binary"
	"math"
)

// Binary wire format:
//   [2 bytes: msg_type LE][payload bytes...]
//
// All numbers are little-endian.
// float32 = 4 bytes, uint32 = 4 bytes, uint64 = 8 bytes, int64 = 8 bytes

// --- Encoder helpers ---

type BufWriter struct {
	Buf []byte
}

func NewBufWriter(capacity int) *BufWriter {
	return &BufWriter{Buf: make([]byte, 0, capacity)}
}

func (w *BufWriter) WriteU16(v uint16) {
	b := [2]byte{}
	binary.LittleEndian.PutUint16(b[:], v)
	w.Buf = append(w.Buf, b[:]...)
}

func (w *BufWriter) WriteU32(v uint32) {
	b := [4]byte{}
	binary.LittleEndian.PutUint32(b[:], v)
	w.Buf = append(w.Buf, b[:]...)
}

func (w *BufWriter) WriteU64(v uint64) {
	b := [8]byte{}
	binary.LittleEndian.PutUint64(b[:], v)
	w.Buf = append(w.Buf, b[:]...)
}

func (w *BufWriter) WriteI64(v int64) {
	w.WriteU64(uint64(v))
}

func (w *BufWriter) WriteF32(v float32) {
	w.WriteU32(math.Float32bits(v))
}

func (w *BufWriter) WriteF64AsF32(v float64) {
	w.WriteF32(float32(v))
}

func (w *BufWriter) WriteBool(v bool) {
	if v {
		w.Buf = append(w.Buf, 1)
	} else {
		w.Buf = append(w.Buf, 0)
	}
}

func (w *BufWriter) WriteBytes(data []byte) {
	w.WriteU32(uint32(len(data)))
	w.Buf = append(w.Buf, data...)
}

func (w *BufWriter) WriteString(s string) {
	w.WriteBytes([]byte(s))
}

// --- Decoder helpers ---

type BufReader struct {
	Data []byte
	Pos  int
}

func NewBufReader(data []byte) *BufReader {
	return &BufReader{Data: data}
}

func (r *BufReader) Remaining() int {
	return len(r.Data) - r.Pos
}

func (r *BufReader) ReadU16() uint16 {
	if r.Remaining() < 2 {
		return 0
	}
	v := binary.LittleEndian.Uint16(r.Data[r.Pos:])
	r.Pos += 2
	return v
}

func (r *BufReader) ReadU32() uint32 {
	if r.Remaining() < 4 {
		return 0
	}
	v := binary.LittleEndian.Uint32(r.Data[r.Pos:])
	r.Pos += 4
	return v
}

func (r *BufReader) ReadU64() uint64 {
	if r.Remaining() < 8 {
		return 0
	}
	v := binary.LittleEndian.Uint64(r.Data[r.Pos:])
	r.Pos += 8
	return v
}

func (r *BufReader) ReadI64() int64 {
	return int64(r.ReadU64())
}

func (r *BufReader) ReadF32() float32 {
	return math.Float32frombits(r.ReadU32())
}

func (r *BufReader) ReadF32AsF64() float64 {
	return float64(r.ReadF32())
}

func (r *BufReader) ReadBool() bool {
	if r.Remaining() < 1 {
		return false
	}
	v := r.Data[r.Pos]
	r.Pos++
	return v != 0
}

func (r *BufReader) ReadBytes() []byte {
	length := int(r.ReadU32())
	if r.Remaining() < length {
		return nil
	}
	v := r.Data[r.Pos : r.Pos+length]
	r.Pos += length
	return v
}

func (r *BufReader) ReadString() string {
	return string(r.ReadBytes())
}

// --- Message encoding ---

func EncodeMessage(msgType int, payload []byte) []byte {
	w := NewBufWriter(2 + len(payload))
	w.WriteU16(uint16(msgType))
	w.Buf = append(w.Buf, payload...)
	return w.Buf
}

func DecodeMessageType(data []byte) int {
	if len(data) < 2 {
		return -1
	}
	return int(binary.LittleEndian.Uint16(data[:2]))
}

func DecodePayload(data []byte) []byte {
	if len(data) < 2 {
		return nil
	}
	return data[2:]
}

// --- Specific message encoders ---

func EncodeHandshakeOK(playerID uint64) []byte {
	w := NewBufWriter(10)
	w.WriteU16(uint16(MsgHandshakeOK))
	w.WriteU64(playerID)
	return w.Buf
}

func EncodePong() []byte {
	w := NewBufWriter(2)
	w.WriteU16(uint16(MsgPong))
	return w.Buf
}

func EncodeMatchFound(matchID string) []byte {
	w := NewBufWriter(32)
	w.WriteU16(uint16(MsgMatchFound))
	w.WriteString(matchID)
	return w.Buf
}

func EncodePlayerSpawned(id uint64, team, class int, x, y float64) []byte {
	w := NewBufWriter(26)
	w.WriteU16(uint16(MsgPlayerSpawned))
	w.WriteU64(id)
	w.WriteU32(uint32(team))
	w.WriteU32(uint32(class))
	w.WriteF64AsF32(x)
	w.WriteF64AsF32(y)
	return w.Buf
}

func EncodePlayerDied(id, killer uint64) []byte {
	w := NewBufWriter(18)
	w.WriteU16(uint16(MsgPlayerDied))
	w.WriteU64(id)
	w.WriteU64(killer)
	return w.Buf
}

func EncodePlayerRespawned(id uint64, x, y float64) []byte {
	w := NewBufWriter(18)
	w.WriteU16(uint16(MsgPlayerRespawned))
	w.WriteU64(id)
	w.WriteF64AsF32(x)
	w.WriteF64AsF32(y)
	return w.Buf
}

func EncodeDamageDealt(target, attacker uint64, amount float64) []byte {
	w := NewBufWriter(22)
	w.WriteU16(uint16(MsgDamageDealt))
	w.WriteU64(target)
	w.WriteU64(attacker)
	w.WriteF64AsF32(amount)
	return w.Buf
}

func EncodeProjectileSpawned(id, owner uint64, team int, x, y, dx, dy, speed, damage float64) []byte {
	w := NewBufWriter(46)
	w.WriteU16(uint16(MsgProjectileSpawned))
	w.WriteU64(id)
	w.WriteU64(owner)
	w.WriteU32(uint32(team))
	w.WriteF64AsF32(x)
	w.WriteF64AsF32(y)
	w.WriteF64AsF32(dx)
	w.WriteF64AsF32(dy)
	w.WriteF64AsF32(speed)
	w.WriteF64AsF32(damage)
	return w.Buf
}

// PlayerSnapshotEntry: 8(id) + 4*5(x,y,vx,vy,hp) + 1(carrying) + 4(seq) = 33 bytes
const PlayerSnapshotSize = 33

func EncodeGameSnapshot(players []PlayerSnapshotData, timer float64) []byte {
	w := NewBufWriter(2 + 4 + 4 + len(players)*PlayerSnapshotSize)
	w.WriteU16(uint16(MsgGameSnapshot))
	w.WriteF64AsF32(timer)
	w.WriteU32(uint32(len(players)))
	for _, p := range players {
		w.WriteU64(p.ID)
		w.WriteF64AsF32(p.X)
		w.WriteF64AsF32(p.Y)
		w.WriteF64AsF32(p.VX)
		w.WriteF64AsF32(p.VY)
		w.WriteF64AsF32(p.HP)
		w.WriteBool(p.Carrying)
		w.WriteU32(uint32(p.Seq))
	}
	return w.Buf
}

type PlayerSnapshotData struct {
	ID       uint64
	X, Y     float64
	VX, VY   float64
	HP       float64
	Carrying bool
	Seq      int
}

func EncodeMapData(width, height, tileSize int, tiles []byte, seed int64) []byte {
	w := NewBufWriter(2 + 4*3 + 4 + len(tiles) + 8)
	w.WriteU16(uint16(MsgMapData))
	w.WriteU32(uint32(width))
	w.WriteU32(uint32(height))
	w.WriteU32(uint32(tileSize))
	w.WriteBytes(tiles)
	w.WriteI64(seed)
	return w.Buf
}

func EncodeRoundStart(round int, duration float64) []byte {
	w := NewBufWriter(10)
	w.WriteU16(uint16(MsgRoundStart))
	w.WriteU32(uint32(round))
	w.WriteF64AsF32(duration)
	return w.Buf
}

func EncodeRoundEnd(winner, round, scoreAlpha, scoreBravo int) []byte {
	w := NewBufWriter(18)
	w.WriteU16(uint16(MsgRoundEnd))
	w.WriteU32(uint32(winner))
	w.WriteU32(uint32(round))
	w.WriteU32(uint32(scoreAlpha))
	w.WriteU32(uint32(scoreBravo))
	return w.Buf
}

func EncodeMatchEnd(winner, scoreAlpha, scoreBravo int) []byte {
	w := NewBufWriter(14)
	w.WriteU16(uint16(MsgMatchEnd))
	w.WriteU32(uint32(winner))
	w.WriteU32(uint32(scoreAlpha))
	w.WriteU32(uint32(scoreBravo))
	return w.Buf
}

func EncodePreGameStart(duration float64) []byte {
	w := NewBufWriter(6)
	w.WriteU16(uint16(MsgPreGameStart))
	w.WriteF64AsF32(duration)
	return w.Buf
}

// --- Specific message decoders (for client->server messages) ---

type InputMoveData struct {
	DX, DY float64
	DT     float64
	Seq    int
}

func DecodeInputMove(payload []byte) InputMoveData {
	r := NewBufReader(payload)
	return InputMoveData{
		DX:  r.ReadF32AsF64(),
		DY:  r.ReadF32AsF64(),
		DT:  r.ReadF32AsF64(),
		Seq: int(r.ReadU32()),
	}
}

type InputShootData struct {
	DX, DY float64
	X, Y   float64
}

func DecodeInputShoot(payload []byte) InputShootData {
	r := NewBufReader(payload)
	return InputShootData{
		DX: r.ReadF32AsF64(),
		DY: r.ReadF32AsF64(),
		X:  r.ReadF32AsF64(),
		Y:  r.ReadF32AsF64(),
	}
}

func DecodeSelectClass(payload []byte) int {
	r := NewBufReader(payload)
	return int(r.ReadU32())
}

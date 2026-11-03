// Package memory はメモリ上に保存する実装。再起動するとデータは消える。
// 最初はこれで動かして、必要になったら firestore に差し替える。
package memory

// TODO: repository.CharacterRepository と repository.MessageRepository を実装する
//   - map と sync.Mutex で十分

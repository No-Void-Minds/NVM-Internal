package auth

import "golang.org/x/crypto/bcrypt"

// dummyHash dipakai saat username tidak ada, supaya waktu respons sama dengan password salah
// dan penyerang tidak bisa menebak username yang terdaftar dari lamanya respons.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("nvm-internal-dummy"), bcrypt.DefaultCost)

func HashPassword(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(h), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func burnPasswordCheck(password string) {
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
}

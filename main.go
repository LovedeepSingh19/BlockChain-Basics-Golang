package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

const difficulty = 2

type Block struct {
	Index      int
	Timestamp  string
	BPM        int
	Hash       string
	PrevHash   string
	Difficulty int
	Nounce     string
}

var BlockChain []Block

type Message struct {
	BPM int
}

var mutex = &sync.Mutex{}

func calculateHash(block Block) string {
	record := strconv.Itoa(block.Index) + block.Timestamp + strconv.Itoa(block.BPM) + block.PrevHash + block.Nounce
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func isHashValid(hash string, difficulty int) bool {
	prefix := strings.Repeat("0", difficulty)
	return strings.HasPrefix(hash, prefix)
}

func generateBlock(oldBlock Block, BPM int) Block {
	var newBlock Block

	t := time.Now()

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.BPM = BPM
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Difficulty = difficulty

	for i := 0; ; i++ {
		hex := fmt.Sprintf("%x", i)
		newBlock.Nounce = hex
		if !isHashValid(calculateHash(newBlock), newBlock.Difficulty) {
			fmt.Println(calculateHash(newBlock), " do more work!!!!")
			time.Sleep(time.Second)
			continue
		} else {
			fmt.Println(calculateHash(newBlock), " work done...!!!!")
			newBlock.Hash = calculateHash(newBlock)
			break
		}
	}

	return newBlock
}

func isBlockValid(newBlock, oldBlock Block) bool {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}
	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}
	if calculateHash(newBlock) != newBlock.Hash {
		return false
	}
	return true
}

func replaceChain(newBlocks []Block) {
	if len(newBlocks) > len(BlockChain) {
		BlockChain = newBlocks
	}
}

func run() error {
	mux := makeMuxRouter()
	httpAddr := os.Getenv(("ADDR"))
	log.Println("Listening on ", os.Getenv("ADDR"))
	s := &http.Server{
		Addr:           ":" + httpAddr,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if err := s.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

func makeMuxRouter() http.Handler {
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
	muxRouter.HandleFunc("/", handleWriteBlock).Methods("POST")
	return muxRouter
}

func handleGetBlockchain(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.MarshalIndent(BlockChain, "", " ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, string(bytes))
}

func handleWriteBlock(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var m Message

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&m); err != nil {
		respondWithJSON(w, r, http.StatusBadRequest, r.Body)
		return
	}
	defer r.Body.Close()

	mutex.Lock()
	newBlock := generateBlock(BlockChain[len(BlockChain)-1], m.BPM)
	mutex.Unlock()

	if isBlockValid(newBlock, BlockChain[len(BlockChain)-1]) {
		BlockChain = append(BlockChain, newBlock)
		spew.Dump(BlockChain)
	}

	respondWithJSON(w, r, http.StatusCreated, newBlock)

}

func respondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	response, err := json.MarshalIndent(payload, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("HTTP 500: Internal Server Error"))
		return
	}
	w.WriteHeader(code)
	w.Write(response)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		t := time.Now()
		genesisBlock := Block{}
		genesisBlock = Block{0, t.String(), 0, calculateHash(genesisBlock), "", difficulty, ""}
		spew.Dump(genesisBlock)

		mutex.Lock()
		BlockChain = append(BlockChain, genesisBlock)
		mutex.Unlock()
	}()
	log.Fatal(run())
}

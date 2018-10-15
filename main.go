package main

import (
  "crypto/sha256"
  "encoding/hex"
  "encoding/json"
  "io"
  "log"
  "net/http"
  "os"
  "time"

  "github.com/gorilla/mux"
  "github.com/joho/godotenv"
)

type Block struct {
  Index      int
  Timestamp  string
  Data       int
  Hash       string
  PrevHash   string
}

type RequestBody struct {
  Data int
}

// WEBSERVER
// *********************************************************************************

func main() {
  err := godotenv.Load()
  if err != nil {
    log.Fatal(err)
  }

  go func() {
    t := time.Now()
    genesisBlock := Block{0, t.String(), 0, "", ""}
    Blockchain = append(Blockchain, genesisBlock)
  }()

  log.Fatal(run())
}

func makeMuxRouter() http.Handler {
  muxRouter := mux.NewRouter()
  muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
  muxRouter.HandleFunc("/", handleWriteBlock).Methods("POST")
  return muxRouter
}

func run() error {
  mux := makeMuxRouter()
  httpAddr := os.Getenv("PORT")
  log.Println("Listening on ", os.Getenv("PORT"))
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

func respondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{}) {
  response, err := json.MarshalIndent(payload, "", "  ")
  if err != nil {
    w.WriteHeader(http.StatusInternalServerError)
    w.Write([]byte("HTTP 500: Internal Server Error"))
    return
  }
  w.WriteHeader(code)
  w.Write(response)
}

// BLOCKCHAIN
// *********************************************************************************

var Blockchain []Block

func handleGetBlockchain(w http.ResponseWriter, r *http.Request) {
  bytes, err := json.MarshalIndent(Blockchain, "", "  ")
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  io.WriteString(w, string(bytes))
}

// Calculate a hash using SHA256 given a block
func calculateHash(b Block) string {
  record := string(b.Index) +b.Timestamp + string(b.Data) + b.PrevHash
  hash := sha256.New()
  hash.Write([]byte(record))
  hashed := hash.Sum(nil)
  return hex.EncodeToString(hashed)
}

// Generate a new block and autoincrement index
func generateBlock(oldBlock Block, data int) (Block, error) {
  var newBlock Block
  t := time.Now()

  newBlock.Index = oldBlock.Index + 1
  newBlock.Timestamp = t.String()
  newBlock.Data = data
  newBlock.PrevHash = oldBlock.Hash
  newBlock.Hash = calculateHash(newBlock)

  return newBlock, nil
}

func isBlockValid(newBlock Block, oldBlock Block) bool {
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


func handleWriteBlock(w http.ResponseWriter, r *http.Request) {
  var requestBody RequestBody
  decoder := json.NewDecoder(r.Body)
  if err := decoder.Decode(&requestBody); err != nil {
    respondWithJSON(w, r, http.StatusBadRequest, r.Body)
    return
  }
  defer r.Body.Close()

  newBlock, err := generateBlock(Blockchain[len(Blockchain)-1], requestBody.Data)

  if err != nil {
    respondWithJSON(w, r, http.StatusInternalServerError, requestBody)
    return
  }

  if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
    newBlockchain := append(Blockchain, newBlock)
    replaceChain(newBlockchain)
  }

  respondWithJSON(w, r, http.StatusCreated, newBlock)

}

func replaceChain(newBlocks []Block) {
  if len(newBlocks) > len(Blockchain) {
    Blockchain = newBlocks
  }
}



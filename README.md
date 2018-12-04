[![GoDoc](https://godoc.org/github.com/rotblauer/go-kf?status.svg)](https://godoc.org/github.com/rotblauer/go-kf)

## go-kf
> A simple key-file data storage library.


```go
import (
  kf "github.com/rotblauer/go-kf"
  "log"
)


type Team struct {
  Name string
  Players []Player
}

type Player struct {
  Name string
  Number int
}

func main() {
  store, err := kf.Newstore(&kf.StoreConfig{
  BaseDir: ".",
  Locking: false, // locking forces mutex-like syncrony per Store, even across instances
  })
  
  if err != nil {
    log.Fatal(err)
  }
  
  cardinals := Team{
    ID: "cards",
    Name: "St. Louis Cardinals",
    Players: []Player{
      {
        Name: "Jan",
        Number: 14,
      },
      {
        Name: "Jeff",
        Number: 42,
      },
    }
  }

  err = store.Set(cardinals.ID)
  if err != nil {
    log.Fatal(err)
  }
  
  for _, p := range cardinals.Players {
    if err := store.Set([]byte(p.Number), cardinals.ID, p.Name); err != nil {
      log.Fatal(err)
    }
  }
 
  keys, err := store.GetKeys(cardinals.ID)
  if err != nil {
    log.Fatal(err)
  }
  
  log.Println("Cardinals players:")
  for _, pf := range keys {
    val, err := store.GetValue(pf)
    if err != nil {
      log.Fatal(err)
    }

    log.Printf("\t %d:%s\n", val, pf)
  }_
}
```



package main

import "fmt"
import "log"
import "mactrix/tsat/truenas_api"

func main() {
    client, err := truenas_api.NewClient("foo", true)
    if err != nil {
        log.Fatal(err)
    }
    jobs := truenas_api.NewJobs(client)
    job := jobs.AddJob(0, "PENDING")
    fmt.Println(job.ID)
}

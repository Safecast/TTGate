/*
 * Testing...
 *
 * Contributors:
 *    Ray Ozzie
 */

package main

import (
	"fmt"
	"time"
)

func main() {

	for i := 0; ; i++ {
		fmt.Printf("Hello from golang..! %d\n", i)
		time.Sleep(1 * time.Second)
	}

}

package main

import (
	"fmt"
	"sync"
	"time"
)

type mensagem struct {
	tipo  int   // 1 = eleição, 2 = falha, 3 = recuperação, 5 = novo líder, 9 = finalizar
	corpo []int // lista de ids participantes ou novo líder
}

var (
	numProcessos = 4
	chans        = make([]chan mensagem, numProcessos)
	status       = make([]bool, numProcessos) // true = ativo, false = falho
	controle     = make(chan int)
	wg           sync.WaitGroup
)

func maxID(ids []int) int {
	max := ids[0]
	for _, id := range ids {
		if id > max {
			max = id
		}
	}
	return max
}

func ElectionStage(id int, in, out chan mensagem) {
	defer wg.Done()
	liderAtual := 0

	for {
		msg := <-in
		time.Sleep(500 * time.Millisecond)

		switch msg.tipo {
		case 1: // eleição
			if !status[id] {
				fmt.Printf("[%d] (falho) repassando eleição\n", id)
				time.Sleep(300 * time.Millisecond)
				out <- msg
				continue
			}

			if len(msg.corpo) > 0 && msg.corpo[0] == id {
				novoLider := maxID(msg.corpo)
				fmt.Printf("[%d] eleição completa. Novo líder: %d\n", id, novoLider)
				time.Sleep(500 * time.Millisecond)
				out <- mensagem{tipo: 5, corpo: []int{novoLider}}
			} else {
				existe := false
				for _, v := range msg.corpo {
					if v == id {
						existe = true
						break
					}
				}
				if !existe {
					msg.corpo = append(msg.corpo, id)
				}
				fmt.Printf("[%d] participando da eleição. Mensagem: %v\n", id, msg.corpo)
				time.Sleep(300 * time.Millisecond)
				out <- msg
			}

		case 2: // falha
			status[id] = false
			fmt.Printf("[%d] entrou em falha\n", id)
			time.Sleep(300 * time.Millisecond)
			controle <- id

		case 3: // recuperação
			status[id] = true
			fmt.Printf("[%d] se recuperou\n", id)
			time.Sleep(300 * time.Millisecond)
			controle <- id

		case 5: // novo líder
			if !status[id] {
				fmt.Printf("[%d] (falho) repassando novo líder\n", id)
				time.Sleep(300 * time.Millisecond)
				out <- msg
				continue
			}
			liderAtual = msg.corpo[0]
			fmt.Printf("[%d] reconheceu novo líder: %d\n", id, liderAtual)
			time.Sleep(300 * time.Millisecond)
			if id != liderAtual {
				out <- msg
			} else {
				controle <- liderAtual // confirmação final
			}

		case 9: // finalizar
			fmt.Printf("[%d] encerrando processo\n", id)
			return

		default:
			fmt.Printf("[%d] mensagem desconhecida: tipo %d\n", id, msg.tipo)
		}
	}
}

func ElectionController() {
	defer wg.Done()

	time.Sleep(1 * time.Second)
	chans[3] <- mensagem{tipo: 2}
	fmt.Println("[Controle] Processo 0 (id=0) falhou")
	<-controle

	time.Sleep(1 * time.Second)
	chans[0] <- mensagem{tipo: 1, corpo: []int{1}}
	fmt.Println("[Controle] Processo 1 inicia eleição")

	lider := <-controle
	fmt.Printf("[Controle] Novo líder reconhecido: %d\n", lider)

	time.Sleep(2 * time.Second)
	chans[(lider+numProcessos-1)%numProcessos] <- mensagem{tipo: 2}
	fmt.Printf("[Controle] Novo líder %d falhou\n", lider)
	<-controle

	time.Sleep(1 * time.Second)
	chans[(lider+1)%numProcessos] <- mensagem{tipo: 1, corpo: []int{(lider + 1) % numProcessos}}
	fmt.Printf("[Controle] Processo %d inicia nova eleição\n", (lider+1)%numProcessos)

	lider = <-controle
	fmt.Printf("[Controle] Novo líder reconhecido: %d\n", lider)

	time.Sleep(1 * time.Second)
	fmt.Println("\n[Controle] Fim da simulação. Encerrando processos...")
	time.Sleep(1 * time.Second)
	for i := 0; i < numProcessos; i++ {
		chans[i] <- mensagem{tipo: 9}
	}
}

func main() {
	for i := 0; i < numProcessos; i++ {
		chans[i] = make(chan mensagem)
		status[i] = true
	}

	wg.Add(numProcessos + 1)
	go ElectionStage(0, chans[3], chans[0]) // este é o lider
	go ElectionStage(1, chans[0], chans[1]) // não é lider, é o processo 0
	go ElectionStage(2, chans[1], chans[2]) // não é lider, é o processo 0
	go ElectionStage(3, chans[2], chans[3]) // não é lider, é o processo 0

	go ElectionController()

	wg.Wait()
}

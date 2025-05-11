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
	status       = make([]bool, numProcessos)
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

	for {
		msg := <-in
		time.Sleep(300 * time.Millisecond)

		switch msg.tipo {
		case 1: // eleição
			if !status[id] {
				fmt.Printf("[%d] (falho) repassando eleição\n", id)
				out <- msg
				continue
			}

			if len(msg.corpo) > 0 && msg.corpo[0] == id {
				novoLider := maxID(msg.corpo)
				fmt.Printf("[%d] eleição completa. Novo líder: %d\n", id, novoLider)
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
				out <- msg
			}

		case 2: // falha
			status[id] = false
			fmt.Printf("[%d] entrou em falha\n", id)
			controle <- id

		case 3: // recuperação
			status[id] = true
			fmt.Printf("[%d] se recuperou\n", id)
			controle <- id

		case 5: // novo líder
			if !status[id] {
				fmt.Printf("[%d] (falho) repassando novo líder\n", id)
				out <- msg
				continue
			}
			lider := msg.corpo[0]
			fmt.Printf("[%d] reconheceu novo líder: %d\n", id, lider)
			if id != lider {
				out <- msg
			} else {
				controle <- lider
			}

		case 9: // finalizar
			fmt.Printf("[%d] encerrando processo\n", id)
			return

		default:
			fmt.Printf("[%d] recebeu tipo desconhecido: %d\n", id, msg.tipo)
		}
	}
}

func ElectionController() {
	defer wg.Done()

	// Início da eleição com processo 0 como líder inicial
	fmt.Println("[Controle] Notificando todos que o líder inicial é o processo 0")
	chans[0] <- mensagem{tipo: 5, corpo: []int{0}}

	lider := <-controle
	fmt.Printf("[Controle] Líder inicial reconhecido: %d\n", lider)

	time.Sleep(2 * time.Second)
	fmt.Println("[Controle] Processo 0 (id=0) falhou")
	chans[3] <- mensagem{tipo: 2}
	<-controle // confirmação da falha

	time.Sleep(2 * time.Second)
	fmt.Println("[Controle] Processo 1 inicia eleição")
	chans[1] <- mensagem{tipo: 1, corpo: []int{1}}

	lider = <-controle
	fmt.Printf("[Controle] Novo líder reconhecido: %d\n", lider)

	time.Sleep(2 * time.Second)
	fmt.Println("[Controle] Processo 0 (id=0) volta a funcionar")
	chans[3] <- mensagem{tipo: 3}
	<-controle // confirmação da recuperação

	time.Sleep(2 * time.Second)
	fmt.Println("[Controle] Processo 3 (id=3) falhou")
	chans[2] <- mensagem{tipo: 2}
	<-controle // confirmação da falha

	time.Sleep(2 * time.Second)
	fmt.Println("[Controle] Processo 2 inicia eleição")
	chans[2] <- mensagem{tipo: 1, corpo: []int{2}}
	lider = <-controle
	fmt.Printf("[Controle] Novo líder reconhecido: %d\n", lider)

	time.Sleep(2 * time.Second)
	fmt.Println("\n[Controle] Fim da simulação. Encerrando processos...")
	for i := 0; i < numProcessos; i++ {
		chans[i] <- mensagem{tipo: 9}
	}
}

func main() {
	for i := 0; i < numProcessos; i++ {
		chans[i] = make(chan mensagem)
		status[i] = true
	}

	// Conectando processos em anel
	wg.Add(numProcessos + 1)
	go ElectionStage(0, chans[3], chans[0])
	go ElectionStage(1, chans[0], chans[1])
	go ElectionStage(2, chans[1], chans[2])
	go ElectionStage(3, chans[2], chans[3])

	go ElectionController()

	wg.Wait()
}

package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Generator генерирует последовательность чисел 1,2,3 и т.д. и
// отправляет их в канал ch. При этом после записи в канал для каждого числа
// вызывается функция fn. Она служит для подсчёта количества и суммы
// сгенерированных чисел.
var wg sync.WaitGroup
var mux sync.Mutex

func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	// 1. Функция Generator
	// ...
	var n int64
	n = 0
	for {
		select {
		case <-ctx.Done():
			close(ch)
			return
		case ch <- n:
			mux.Lock()
			fn(n)
			mux.Unlock()
			n += 1

		}
	}

}

// Worker читает число из канала in и пишет его в канал out.
func Worker(in <-chan int64, out chan<- int64) {
	// 2. Функция Worker
	defer wg.Done()
	for {
		// получаем числа из канала
		v, ok := <-in
		if !ok {
			close(out)
			break
		}
		// отправляем результат в другой канал
		out <- v
	}
}

func main() {
	chIn := make(chan int64)

	// 3. Создание контекста
	// ...
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	// для проверки будем считать количество и сумму отправленных чисел
	var inputSum int64   // сумма сгенерированных чисел
	var inputCount int64 // количество сгенерированных чисел

	// генерируем числа, считая параллельно их количество и сумму
	go Generator(ctx, chIn, func(i int64) {
		inputSum += i
		inputCount++
	})

	const NumOut = 10 // количество обрабатывающих горутин и каналов
	// outs — слайс каналов, куда будут записываться числа из chIn
	outs := make([]chan int64, NumOut)
	for i := 0; i < NumOut; i++ {
		// создаём каналы и для каждого из них вызываем горутину Worker
		wg.Add(1)
		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}

	// amounts — слайс, в который собирается статистика по горутинам
	amounts := make([]int64, NumOut)
	// chOut — канал, в который будут отправляться числа из горутин `outs[i]`
	chOut := make(chan int64, NumOut)

	// 4. Собираем числа из каналов outs
	// ...
	for i := 0; i < NumOut; i++ {
		// Собираем числа из каналов outs
		wg.Add(1)
		go func(in <-chan int64, i int64) {
			for {
				v, ok := <-in
				if !ok {
					wg.Done()
					break
				}
				chOut <- v
				amounts[i]++
			}
		}(outs[i], int64(i))

	}
	go func() {
		// ждём завершения работы всех горутин для outs
		wg.Wait()
		// закрываем результирующий канал
		close(chOut)
	}()

	var count int64 // количество чисел результирующего канала
	var sum int64   // сумма чисел результирующего канала

	// 5. Читаем числа из результирующего канала
	for v := range chOut {
		sum += v
		count++
	}

	fmt.Println("Количество чисел", inputCount, count)
	fmt.Println("Сумма чисел", inputSum, sum)
	fmt.Println("Разбивка по каналам", amounts)

	// проверка результатов
	if inputSum != sum {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum, sum)
	}
	if inputCount != count {
		log.Fatalf("Ошибка: количество чисел не равно: %d != %d\n", inputCount, count)
	}
	for _, v := range amounts {
		inputCount -= v
	}
	if inputCount != 0 {
		log.Fatalf("Ошибка: разделение чисел по каналам неверное\n")
	}
}

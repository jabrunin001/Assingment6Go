package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/mat"
)

func main() {
	// Load data from CSV file
	f, err := os.Open("boston.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	if _, err := r.Read(); err != nil {
		log.Fatal(err)
	}
	records, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	// Define indices of features to be used for prediction
	featureIndices := []int{1, 5, 12, 10}
	responseIndex := len(records[0]) - 1

	// Parse data into matrix
	n := len(records)
	p := len(featureIndices)
	x := mat.NewDense(n, p+1, nil) // +1 for intercept
	y := mat.NewDense(n, 1, nil)

	for i, record := range records {
		x.Set(i, 0, 1.0) // Intercept
		for j, featureIndex := range featureIndices {
			val := parseFloat(record[featureIndex])
			x.Set(i, j+1, val)
		}
		y.Set(i, 0, parseFloat(record[responseIndex]))
	}

	// Split data into training and testing sets
	trainFrac := 0.7
	nTrain := int(float64(n) * trainFrac)
	xTrain := x.Slice(0, nTrain, 0, p+1).(*mat.Dense)
	yTrain := y.Slice(0, nTrain, 0, 1).(*mat.Dense)
	xTest := x.Slice(nTrain, n, 0, p+1).(*mat.Dense)
	yTest := y.Slice(nTrain, n, 0, 1).(*mat.Dense)

	// Prepare channels and wait group for concurrency
	modelsCh := make(chan *mat.Dense, 2)
	mseCh := make(chan float64, 2)
	var wg sync.WaitGroup

	// Train and test models concurrently
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			model := trainLinearRegression(xTrain, yTrain)
			yPred := predictLinearRegression(model, xTest)
			mse := computeMSE(yTest, yPred)
			modelsCh <- model
			mseCh <- mse
		}()
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(modelsCh)
	close(mseCh)

	// Retrieve results from channels
	mse1 := <-mseCh
	mse2 := <-mseCh

	// Print mean squared error for each model
	fmt.Printf("Model 1 MSE: %f\n", mse1)
	fmt.Printf("Model 2 MSE: %f\n", mse2)

}

func trainLinearRegression(x, y *mat.Dense) *mat.Dense {
	var xTx, xTy, beta mat.Dense
	xt := x.T()
	xTx.Mul(xt, x)
	xTy.Mul(xt, y)

	var xTxInv mat.Dense
	if err := xTxInv.Inverse(&xTx); err != nil {
		log.Fatal(err)
	}

	beta.Mul(&xTxInv, &xTy)
	return &beta
}

func predictLinearRegression(beta *mat.Dense, x *mat.Dense) *mat.Dense {
	var yPred mat.Dense
	yPred.Mul(x, beta)
	return &yPred
}

func computeMSE(yTrue, yPred *mat.Dense) float64 {
	r, _ := yTrue.Dims()
	diff := mat.NewDense(r, 1, nil)
	diff.Sub(yTrue, yPred)

	diffVec := mat.NewVecDense(r, diff.RawMatrix().Data)
	squaredDiff := mat.NewVecDense(r, nil)
	squaredDiff.MulElemVec(diffVec, diffVec)

	mse := floats.Sum(squaredDiff.RawVector().Data) / float64(r)
	return mse
}

func parseFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Fatal(err)
	}
	return f
}

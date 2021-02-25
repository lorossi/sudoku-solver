# sudoku solver

A sudoku solver made in Go, supporting text and image input

## Sample input and output

### Image

#### Sample input

![unsolved-sudoku](sample/sudoku-1.png)

*[courtesy of sudoku.com](https://sudoku.com/)

#### Sample outputs

- Text output:

```plaintext
-------------
|921|538|476|
|473|162|859|
|568|479|213|
-------------
|856|913|742|
|234|687|591|
|719|254|368|
-------------
|347|891|625|
|185|726|934|
|692|345|187|
-------------
```

- Image output:

![solved-sudoku](sample/solved-1.png)

- File output: *(sample/solved-1.txt)*

`921538476473162859568479213856913742234687591719254368347891625185726934692345187`

### String

#### Sample input

String: `--------8---29-65-1---73----31-----4---38----82----1---9-5-7---2------7------49--`

#### Sample outputs

- Text output

```plaintext
-------------
|962|145|738|
|374|298|651|
|185|673|249|
-------------
|531|762|894|
|649|381|527|
|827|459|163|
-------------
|496|517|382|
|218|936|475|
|753|824|916|
-------------
```

- Image output:

![solved-sudoku](sample/solved-2.png)

- File output: *(sample/solved-2.txt)*

`962145738374298651185673249531762894649381527827459163496517382218936475753824916`

### File

#### Sample input

File *(sample/sudoku-3.txt)*

`-9---7----------457---8---6-----53---8------9--49---6-5-16---------4--1-------257`


#### Sample outputs

- Text output

```plaintext
-------------
|496|157|832|
|218|396|745|
|753|284|196|
-------------
|962|415|378|
|185|763|429|
|374|928|561|
-------------
|531|672|984|
|827|549|613|
|649|831|257|
-------------
```

- Image output:

![solved-sudoku](sample/solved-3.png)

- File output: *(sample/solved-3.txt)*

`496157832218396745753284196962415378185763429374928561531672984827549613649831257`
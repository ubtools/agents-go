interface Currency {
  code: string
}

type UUID = bigint

interface Tx {
  id: UUID
  currency: Currency
  amount: bigint
  externalCurrency: bigint
  externalAmount: bigint
  fee: bigint
  status: string
  parent?: Tx
  previous?: Tx
  children: Tx[]
  createdAt: Date
  finishedAt?: Date
}

interface SrcTx extends Tx {
   
}

type order = Tx[]
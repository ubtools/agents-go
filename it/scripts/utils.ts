export async function arrayFromAsync<T>(i: AsyncIterable<T>): Promise<Array<T>> {
  const a = [];
  for await (const x of i) {
    a.push(x);
  }
  return a;
}
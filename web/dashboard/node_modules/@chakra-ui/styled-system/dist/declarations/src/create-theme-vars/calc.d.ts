export declare type Operand = string | number | {
    reference: string;
};
interface CalcChain {
    add: (...operands: Array<Operand>) => CalcChain;
    subtract: (...operands: Array<Operand>) => CalcChain;
    multiply: (...operands: Array<Operand>) => CalcChain;
    divide: (...operands: Array<Operand>) => CalcChain;
    negate: () => CalcChain;
    toString: () => string;
}
export declare const calc: ((x: Operand) => CalcChain) & {
    add: (...operands: Array<Operand>) => string;
    subtract: (...operands: Array<Operand>) => string;
    multiply: (...operands: Array<Operand>) => string;
    divide: (...operands: Array<Operand>) => string;
    negate: (x: Operand) => string;
};
export {};
//# sourceMappingURL=calc.d.ts.map
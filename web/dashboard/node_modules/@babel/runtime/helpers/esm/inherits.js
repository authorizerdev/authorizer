import setPrototypeOf from "./setPrototypeOf.js";
export default function _inherits(subClass, superClass) {
  if (typeof superClass !== "function" && superClass !== null) {
    throw new TypeError("Super expression must either be null or a function");
  }

  Object.defineProperty(subClass, "prototype", {
    value: Object.create(superClass && superClass.prototype, {
      constructor: {
        value: subClass,
        writable: true,
        configurable: true
      }
    }),
    writable: false
  });
  if (superClass) setPrototypeOf(subClass, superClass);
}
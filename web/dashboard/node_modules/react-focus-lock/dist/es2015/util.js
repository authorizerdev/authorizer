export function deferAction(action) {
  setTimeout(action, 1);
}
export var inlineProp = function inlineProp(name, value) {
  var obj = {};
  obj[name] = value;
  return obj;
};
export var extractRef = function extractRef(ref) {
  return ref && 'current' in ref ? ref.current : ref;
};
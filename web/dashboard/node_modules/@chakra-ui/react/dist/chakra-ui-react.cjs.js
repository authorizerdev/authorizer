'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-react.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-react.cjs.dev.js");
}

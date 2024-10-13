'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-number-input.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-number-input.cjs.dev.js");
}

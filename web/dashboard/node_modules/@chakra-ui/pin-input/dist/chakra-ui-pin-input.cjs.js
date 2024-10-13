'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-pin-input.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-pin-input.cjs.dev.js");
}

'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-radio.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-radio.cjs.dev.js");
}

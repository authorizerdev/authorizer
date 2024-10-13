'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-select.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-select.cjs.dev.js");
}

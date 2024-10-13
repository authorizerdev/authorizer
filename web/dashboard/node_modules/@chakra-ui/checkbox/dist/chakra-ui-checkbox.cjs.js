'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-checkbox.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-checkbox.cjs.dev.js");
}

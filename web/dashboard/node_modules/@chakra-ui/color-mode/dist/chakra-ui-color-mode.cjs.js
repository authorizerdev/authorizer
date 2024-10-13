'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-color-mode.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-color-mode.cjs.dev.js");
}

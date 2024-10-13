'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-focus-lock.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-focus-lock.cjs.dev.js");
}

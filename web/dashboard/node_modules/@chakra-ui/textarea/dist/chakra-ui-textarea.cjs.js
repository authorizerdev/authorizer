'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-textarea.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-textarea.cjs.dev.js");
}

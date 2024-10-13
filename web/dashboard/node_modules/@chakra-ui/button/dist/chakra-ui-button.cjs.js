'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-button.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-button.cjs.dev.js");
}

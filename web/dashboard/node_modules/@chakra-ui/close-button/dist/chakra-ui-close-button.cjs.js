'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-close-button.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-close-button.cjs.dev.js");
}

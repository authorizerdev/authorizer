'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-modal.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-modal.cjs.dev.js");
}

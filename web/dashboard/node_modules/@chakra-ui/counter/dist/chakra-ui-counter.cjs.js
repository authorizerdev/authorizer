'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-counter.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-counter.cjs.dev.js");
}

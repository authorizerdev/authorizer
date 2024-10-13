'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-transition.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-transition.cjs.dev.js");
}

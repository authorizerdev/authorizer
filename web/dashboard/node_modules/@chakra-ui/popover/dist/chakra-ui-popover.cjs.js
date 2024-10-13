'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-popover.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-popover.cjs.dev.js");
}

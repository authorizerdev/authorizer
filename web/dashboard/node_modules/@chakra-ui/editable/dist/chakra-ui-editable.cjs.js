'use strict';

if (process.env.NODE_ENV === "production") {
  module.exports = require("./chakra-ui-editable.cjs.prod.js");
} else {
  module.exports = require("./chakra-ui-editable.cjs.dev.js");
}

import { anatomy } from '@chakra-ui/theme-tools';

/**
 * **Accordion anatomy**
 * - Item: the accordion item contains the button and panel
 * - Button: the button is the trigger for the panel
 * - Panel: the panel is the content of the accordion item
 * - Icon: the expanded/collapsed icon
 */

var accordionAnatomy = anatomy("accordion").parts("container", "item", "button", "panel").extend("icon");
/**
 * **Alert anatomy**
 * - Title: the alert's title
 * - Description: the alert's description
 * - Icon: the alert's icon
 */

var alertAnatomy = anatomy("alert").parts("title", "description", "container").extend("icon");
/**
 * **Avatar anatomy**
 * - Container: the container for the avatar
 * - Label: the avatar initials text
 * - Excess Label: the label or text that represents excess avatar count.
 * Typically used in avatar groups.
 * - Group: the container for the avatar group
 */

var avatarAnatomy = anatomy("avatar").parts("label", "badge", "container").extend("excessLabel", "group");
/**
 * **Breadcrumb anatomy**
 * - Item: the container for a breadcrumb item
 * - Link: the element that represents the breadcrumb link
 * - Container: the container for the breadcrumb items
 * - Separator: the separator between breadcrumb items
 */

var breadcrumbAnatomy = anatomy("breadcrumb").parts("link", "item", "container").extend("separator");
var buttonAnatomy = anatomy("button").parts();
var checkboxAnatomy = anatomy("checkbox").parts("control", "icon", "container").extend("label");
var circularProgressAnatomy = anatomy("progress").parts("track", "filledTrack").extend("label");
var drawerAnatomy = anatomy("drawer").parts("overlay", "dialogContainer", "dialog").extend("header", "closeButton", "body", "footer");
var editableAnatomy = anatomy("editable").parts("preview", "input");
var formAnatomy = anatomy("form").parts("container", "requiredIndicator", "helperText");
var formErrorAnatomy = anatomy("formError").parts("text", "icon");
var inputAnatomy = anatomy("input").parts("addon", "field", "element");
var listAnatomy = anatomy("list").parts("container", "item", "icon");
var menuAnatomy = anatomy("menu").parts("button", "list", "item").extend("groupTitle", "command", "divider");
var modalAnatomy = anatomy("modal").parts("overlay", "dialogContainer", "dialog").extend("header", "closeButton", "body", "footer");
var numberInputAnatomy = anatomy("numberinput").parts("root", "field", "stepperGroup", "stepper");
var pinInputAnatomy = anatomy("pininput").parts("field");
var popoverAnatomy = anatomy("popover").parts("content", "header", "body", "footer").extend("popper", "arrow", "closeButton");
var progressAnatomy = anatomy("progress").parts("label", "filledTrack", "track");
var radioAnatomy = anatomy("radio").parts("container", "control", "label");
var selectAnatomy = anatomy("select").parts("field", "icon");
var sliderAnatomy = anatomy("slider").parts("container", "track", "thumb", "filledTrack");
var statAnatomy = anatomy("stat").parts("container", "label", "helpText", "number", "icon");
var switchAnatomy = anatomy("switch").parts("container", "track", "thumb");
var tableAnatomy = anatomy("table").parts("table", "thead", "tbody", "tr", "th", "td", "tfoot", "caption");
var tabsAnatomy = anatomy("tabs").parts("root", "tab", "tablist", "tabpanel", "tabpanels", "indicator");
/**
 * **Tag anatomy**
 * - Container: the container for the tag
 * - Label: the text content of the tag
 * - closeButton: the close button for the tag
 */

var tagAnatomy = anatomy("tag").parts("container", "label", "closeButton");

export { accordionAnatomy, alertAnatomy, avatarAnatomy, breadcrumbAnatomy, buttonAnatomy, checkboxAnatomy, circularProgressAnatomy, drawerAnatomy, editableAnatomy, formAnatomy, formErrorAnatomy, inputAnatomy, listAnatomy, menuAnatomy, modalAnatomy, numberInputAnatomy, pinInputAnatomy, popoverAnatomy, progressAnatomy, radioAnatomy, selectAnatomy, sliderAnatomy, statAnatomy, switchAnatomy, tableAnatomy, tabsAnatomy, tagAnatomy };

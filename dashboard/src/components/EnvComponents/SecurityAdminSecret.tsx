import React from "react";
import {
  Flex,
  Stack,
  Center,
  Text,
  Input,
  InputGroup,
  InputRightElement,
  useMediaQuery,
} from "@chakra-ui/react";
import { FaRegEyeSlash, FaRegEye } from "react-icons/fa";
import InputField from "../InputField";
import { HiddenInputType } from "../../constants";
const SecurityAdminSecret = ({
  variables,
  setVariables,
  fieldVisibility,
  setFieldVisibility,
  validateAdminSecretHandler,
  adminSecret,
}: any) => {
  const [isNotSmallerScreen] = useMediaQuery("(min-width:600px)");
  return (
    <div>
      {" "}
      <Text fontSize="md" paddingTop="2%" fontWeight="bold">
        Security (Admin Secret)
      </Text>
      <Stack
        spacing={6}
        padding="0 5%"
        marginTop="3%"
        border="1px solid #ff7875"
        borderRadius="5px"
      >
        <Flex
          marginTop={isNotSmallerScreen ? "3%" : "5%"}
          direction={isNotSmallerScreen ? "row" : "column"}
        >
          <Flex
            mt={3}
            w={isNotSmallerScreen ? "30%" : "40%"}
            justifyContent="start"
            alignItems="center"
          >
            <Text fontSize="sm">Old Admin Secret:</Text>
          </Flex>
          <Center
            w={isNotSmallerScreen ? "70%" : "100%"}
            mt={isNotSmallerScreen ? "0" : "3"}
          >
            <InputGroup size="sm">
              <Input
                borderRadius={5}
                size="sm"
                placeholder="Enter Old Admin Secret"
                value={adminSecret.value as string}
                onChange={(event: any) => validateAdminSecretHandler(event)}
                type={
                  !fieldVisibility[HiddenInputType.OLD_ADMIN_SECRET]
                    ? "password"
                    : "text"
                }
              />
              <InputRightElement
                right="5px"
                children={
                  <Flex>
                    {fieldVisibility[HiddenInputType.OLD_ADMIN_SECRET] ? (
                      <Center
                        w="25px"
                        margin="0 1.5%"
                        cursor="pointer"
                        onClick={() =>
                          setFieldVisibility({
                            ...fieldVisibility,
                            [HiddenInputType.OLD_ADMIN_SECRET]: false,
                          })
                        }
                      >
                        <FaRegEyeSlash color="#bfbfbf" />
                      </Center>
                    ) : (
                      <Center
                        w="25px"
                        margin="0 1.5%"
                        cursor="pointer"
                        onClick={() =>
                          setFieldVisibility({
                            ...fieldVisibility,
                            [HiddenInputType.OLD_ADMIN_SECRET]: true,
                          })
                        }
                      >
                        <FaRegEye color="#bfbfbf" />
                      </Center>
                    )}
                  </Flex>
                }
              />
            </InputGroup>
          </Center>
        </Flex>
        <Flex
          paddingBottom="3%"
          direction={isNotSmallerScreen ? "row" : "column"}
        >
          <Flex
            w={isNotSmallerScreen ? "30%" : "50%"}
            justifyContent="start"
            alignItems="center"
          >
            <Text fontSize="sm">New Admin Secret:</Text>
          </Flex>
          <Center
            w={isNotSmallerScreen ? "70%" : "100%"}
            mt={isNotSmallerScreen ? "0" : "3"}
          >
            <InputField
              borderRadius={5}
              mb={3}
              variables={variables}
              setVariables={setVariables}
              inputType={HiddenInputType.ADMIN_SECRET}
              fieldVisibility={fieldVisibility}
              setFieldVisibility={setFieldVisibility}
              isDisabled={adminSecret.disableInputField}
              placeholder="Enter New Admin Secret"
            />
          </Center>
        </Flex>
      </Stack>
    </div>
  );
};

export default SecurityAdminSecret;
import React from "react";
import {
  Box,
  Divider,
  Flex,
  Stack,
  Center,
  Text,
  Button,
  Input,
  InputGroup,
  InputRightElement,
  useToast,
} from "@chakra-ui/react";
import { FaRegEyeSlash, FaRegEye } from "react-icons/fa";
import InputField from "../../InputField";
import { TextInputType, HiddenInputType } from "../../../constants";
const DangerArea = ({
  variables,
  setVariables,
  fieldVisibility,
  setFieldVisibility,
  validateAdminSecretHandler,
  adminSecret,
}: any) => {
  return (
    <div>
      {" "}
      <Text fontSize="md" paddingTop="2%" fontWeight="bold">
        Danger
      </Text>
      <Stack
        spacing={6}
        padding="0 5%"
        marginTop="3%"
        border="1px solid #ff7875"
        borderRadius="5px"
      >
        <Stack spacing={6} padding="3% 0">
          <Text fontStyle="italic" fontSize="sm" color="gray.600">
            Note: Database related environment variables cannot be updated from
            dashboard :(
          </Text>
          <Flex>
            <Flex w="30%" justifyContent="start" alignItems="center">
              <Text fontSize="sm">DataBase Name:</Text>
            </Flex>
            <Center w="70%">
              <InputField
                variables={variables}
                setVariables={setVariables}
                inputType={TextInputType.DATABASE_NAME}
                isDisabled={true}
              />
            </Center>
          </Flex>
          <Flex>
            <Flex w="30%" justifyContent="start" alignItems="center">
              <Text fontSize="sm">DataBase Type:</Text>
            </Flex>
            <Center w="70%">
              <InputField
                variables={variables}
                setVariables={setVariables}
                inputType={TextInputType.DATABASE_TYPE}
                isDisabled={true}
              />
            </Center>
          </Flex>
          <Flex>
            <Flex w="30%" justifyContent="start" alignItems="center">
              <Text fontSize="sm">DataBase URL:</Text>
            </Flex>
            <Center w="70%">
              <InputField
                variables={variables}
                setVariables={setVariables}
                inputType={TextInputType.DATABASE_URL}
                isDisabled={true}
              />
            </Center>
          </Flex>
        </Stack>
        <Flex marginTop="3%">
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Old Admin Secret:</Text>
          </Flex>
          <Center w="70%">
            <InputGroup size="sm">
              <Input
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
        <Flex paddingBottom="3%">
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">New Admin Secret:</Text>
          </Flex>
          <Center w="70%">
            <InputField
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

export default DangerArea;

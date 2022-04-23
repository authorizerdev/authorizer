import React from "react";
import {
  Button,
  Center,
  Flex,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  useDisclosure,
  Text,
  useToast,
  Input,
  Spinner,
} from "@chakra-ui/react";
import { useClient } from "urql";
import { FaSave } from "react-icons/fa";
import {
  ECDSAEncryptionType,
  HMACEncryptionType,
  RSAEncryptionType,
  SelectInputType,
  TextAreaInputType,
} from "../constants";
import InputField from "./InputField";
import { GenerateKeys, UpdateEnvVariables } from "../graphql/mutation";

interface propTypes {
  jwtType: string;
  getData: Function;
}

interface stateVarTypes {
  JWT_TYPE: string;
  JWT_SECRET: string;
  JWT_PRIVATE_KEY: string;
  JWT_PUBLIC_KEY: string;
}

const initState: stateVarTypes = {
  JWT_TYPE: "",
  JWT_SECRET: "",
  JWT_PRIVATE_KEY: "",
  JWT_PUBLIC_KEY: "",
};

const GenerateKeysModal = ({ jwtType, getData }: propTypes) => {
  const client = useClient();
  const toast = useToast();
  const { isOpen, onOpen, onClose } = useDisclosure();
  const [stateVariables, setStateVariables] = React.useState<stateVarTypes>({
    ...initState,
  });
  const [isLoading, setIsLoading] = React.useState(false);

  React.useEffect(() => {
    if (isOpen) {
      setStateVariables({ ...initState, JWT_TYPE: jwtType });
    }
  }, [isOpen]);

  const fetchKeys = async () => {
    setIsLoading(true);
    try {
      const res = await client
        .mutation(GenerateKeys, { params: { type: stateVariables.JWT_TYPE } })
        .toPromise();
      if (res?.error) {
        toast({
          title: "Error occurred generating jwt keys",
          isClosable: true,
          status: "error",
          position: "bottom-right",
        });
        closeHandler();
      } else {
        setStateVariables({
          ...stateVariables,
          JWT_SECRET: res?.data?._generate_jwt_keys?.secret || "",
          JWT_PRIVATE_KEY: res?.data?._generate_jwt_keys?.private_key || "",
          JWT_PUBLIC_KEY: res?.data?._generate_jwt_keys?.public_key || "",
        });
      }
    } catch (error) {
      console.log(error);
    } finally {
      setIsLoading(false);
    }
  };

  React.useEffect(() => {
    if (isOpen && stateVariables.JWT_TYPE) {
      fetchKeys();
    }
  }, [stateVariables.JWT_TYPE]);

  const saveHandler = async () => {
    const res = await client
      .mutation(UpdateEnvVariables, { params: { ...stateVariables } })
      .toPromise();

    if (res.error) {
      toast({
        title: "Error occurred setting jwt keys",
        isClosable: true,
        status: "error",
        position: "bottom-right",
      });

      return;
    }
    toast({
      title: "JWT keys updated successfully",
      isClosable: true,
      status: "success",
      position: "bottom-right",
    });
    closeHandler();
  };

  const closeHandler = () => {
    setStateVariables({ ...initState });
    getData();
    onClose();
  };

  return (
    <>
      <Button
        colorScheme="blue"
        h="1.75rem"
        size="sm"
        variant="ghost"
        onClick={onOpen}
      >
        Generate new keys
      </Button>
      <Modal isOpen={isOpen} onClose={closeHandler}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>New JWT keys</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Flex>
              <Flex w="30%" justifyContent="start" alignItems="center">
                <Text fontSize="sm">JWT Type:</Text>
              </Flex>
              <InputField
                variables={stateVariables}
                setVariables={setStateVariables}
                inputType={SelectInputType.JWT_TYPE}
                value={SelectInputType.JWT_TYPE}
                options={{
                  ...HMACEncryptionType,
                  ...RSAEncryptionType,
                  ...ECDSAEncryptionType,
                }}
              />
            </Flex>
            {isLoading ? (
              <Center minH="25vh">
                <Spinner />
              </Center>
            ) : (
              <>
                {Object.values(HMACEncryptionType).includes(
                  stateVariables.JWT_TYPE
                ) ? (
                  <Flex marginTop="8">
                    <Flex w="23%" justifyContent="start" alignItems="center">
                      <Text fontSize="sm">JWT Secret</Text>
                    </Flex>
                    <Center w="77%">
                      <Input
                        size="sm"
                        value={stateVariables.JWT_SECRET}
                        onChange={(event: any) =>
                          setStateVariables({
                            ...stateVariables,
                            JWT_SECRET: event.target.value,
                          })
                        }
                        readOnly
                      />
                    </Center>
                  </Flex>
                ) : (
                  <>
                    <Flex marginTop="8">
                      <Flex w="23%" justifyContent="start" alignItems="center">
                        <Text fontSize="sm">Public Key</Text>
                      </Flex>
                      <Center w="77%">
                        <InputField
                          variables={stateVariables}
                          setVariables={setStateVariables}
                          inputType={TextAreaInputType.JWT_PUBLIC_KEY}
                          placeholder="Add public key here"
                          minH="25vh"
                          readOnly
                        />
                      </Center>
                    </Flex>
                    <Flex marginTop="8">
                      <Flex w="23%" justifyContent="start" alignItems="center">
                        <Text fontSize="sm">Private Key</Text>
                      </Flex>
                      <Center w="77%">
                        <InputField
                          variables={stateVariables}
                          setVariables={setStateVariables}
                          inputType={TextAreaInputType.JWT_PRIVATE_KEY}
                          placeholder="Add private key here"
                          minH="25vh"
                          readOnly
                        />
                      </Center>
                    </Flex>
                  </>
                )}
              </>
            )}
          </ModalBody>

          <ModalFooter>
            <Button
              leftIcon={<FaSave />}
              colorScheme="blue"
              variant="solid"
              onClick={saveHandler}
              isDisabled={isLoading}
            >
              <Center h="100%" pt="5%">
                Apply
              </Center>
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </>
  );
};

export default GenerateKeysModal;

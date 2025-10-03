import amqp from "amqplib";
import logger from "./logger.js";

let connection = null;
let channel = null;

const RABBITMQ_URL =
  process.env.RABBITMQ_URL || "amqp://admin:admin@localhost:5672";
const EXCHANGE_NAME = "auth_events";

export async function connectRabbitMQ() {
  try {
    connection = await amqp.connect(RABBITMQ_URL);
    channel = await connection.createChannel();

    // Create exchange for auth events
    await channel.assertExchange(EXCHANGE_NAME, "topic", { durable: true });

    logger.info("Connected to RabbitMQ");

    connection.on("error", (err) => {
      logger.error(`RabbitMQ connection error: ${err.message}`, {
        stack: err.stack,
      });
    });

    connection.on("close", () => {
      logger.warn("RabbitMQ connection closed. Reconnecting in 5s...");
      setTimeout(connectRabbitMQ, 5000);
    });

    return { connection, channel };
  } catch (error) {
    logger.error(`Failed to connect to RabbitMQ: ${error.message}`, {
      stack: error.stack,
    });
    setTimeout(connectRabbitMQ, 5000);
  }
}

export async function publishEvent(routingKey, message) {
  if (!channel) {
    logger.error("RabbitMQ channel not initialized");
    return false;
  }

  try {
    const messageBuffer = Buffer.from(JSON.stringify(message));
    channel.publish(EXCHANGE_NAME, routingKey, messageBuffer, {
      persistent: true,
      contentType: "application/json",
    });
    logger.debug(`Published event: ${routingKey}`, message);
    return true;
  } catch (error) {
    logger.error(`Failed to publish message: ${error.message}`, {
      stack: error.stack,
    });
    return false;
  }
}

export function getChannel() {
  return channel;
}

export async function closeRabbitMQ() {
  try {
    await channel?.close();
    await connection?.close();
    logger.info("RabbitMQ connection closed");
  } catch (error) {
    logger.error(`Error closing RabbitMQ connection: ${error.message}`, {
      stack: error.stack,
    });
  }
}

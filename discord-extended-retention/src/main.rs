use anyhow::anyhow;
use serde::Deserialize;
use serenity::async_trait;
use serenity::prelude::*;
use serenity::{http::CacheHttp, Client};
use std::collections::HashMap;
use std::sync::Arc;

#[derive(Deserialize, Debug)]
struct ChannelConfig {
    retention: String,
}

#[derive(Deserialize, Debug)]
struct ConfigSer {
    token: String,
    application_id: u64,
    channels: HashMap<String, ChannelConfig>,
}

struct Config {
    token: String,
    application_id: u64,
    channels: HashMap<u64, ChannelConfig>,
}

impl TryFrom<ConfigSer> for Config {
    type Error = anyhow::Error;
    fn try_from(c: ConfigSer) -> Result<Self, Self::Error> {
        Ok(Self {
            token: c.token,
            application_id: c.application_id,
            channels: c
                .channels
                .into_iter()
                .map(|(k, v)| -> Result<(u64, ChannelConfig), anyhow::Error> {
                    Ok((
                        k.parse::<u64>()
                            .map_err(|_| anyhow!("could not parse channel ID '{}' in config", k))?
                            .into(),
                        v,
                    ))
                })
                .collect::<Result<HashMap<u64, ChannelConfig>, anyhow::Error>>()?,
        })
    }
}

struct ConfigKey();

impl TypeMapKey for ConfigKey {
    type Value = &'static Config;
}

struct Control {
    shard_manager: Arc<Mutex<serenity::client::bridge::gateway::ShardManager>>,
}

struct ControlKey();

impl TypeMapKey for ControlKey {
    type Value = Control;
}

fn load_config() -> Result<Config, anyhow::Error> {
    let contents = std::fs::read_to_string("Config.toml")?;
    let toml_native: toml::value::Value = toml::from_str(&contents)?;
    println!("toml: {:?}", toml_native);

    let conf: ConfigSer = toml::from_str(&contents)?;
    println!("{:?}", conf);
    Ok(Config::try_from(conf)?)
}

struct StartupHandler;

#[async_trait]
impl EventHandler for StartupHandler {
    async fn ready(&self, ctx: Context, _data_about_bot: serenity::model::prelude::Ready) -> () {
        println!("Ready");
        let data = ctx.data.read().await;
        let config = data.get::<ConfigKey>().unwrap();

        for (channel, retention) in config.channels.iter() {
            let result = ctx
                .http()
                .get_messages(*channel, "")
                .await;
            match result {
                Ok(messages) => {
                    let _ = &retention.retention;
                    println!("Messages: {:?}", messages.iter().map(|m| (&m.author.name, m.id)).collect::<Vec<_>>());
                }
                Err(err) => {
                    println!("could not fetch from {}: {}", channel, err);
                }
            }
        }

        let arcdata = ctx.data.clone();
        tokio::spawn(async move {
            println!("Done, shutting down");
            let data = arcdata.read().await;
            let control = data.get::<ControlKey>().unwrap();
            let mut shard_manager = control.shard_manager.lock().await;
            shard_manager.shutdown_all().await;
        });
    }
}

#[tokio::main(flavor = "current_thread")]
async fn main() -> Result<(), anyhow::Error> {
    let config = Box::leak(Box::new(load_config()?));

    let mut client = Client::builder(&config.token, GatewayIntents::empty())
        .type_map_insert::<ConfigKey>(&*config)
        .event_handler(StartupHandler)
        .application_id(config.application_id)
        .await?;

    {
        let shard_manager = client.shard_manager.clone();
        let mut data = client.data.write().await;
        data.insert::<ControlKey>(Control { shard_manager });
    }

    println!("Starting");
    if let Err(why) = client.start().await {
        println!("Err with client: {:?}", why);
    }
    Ok(())
}

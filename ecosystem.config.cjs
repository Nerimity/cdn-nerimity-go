module.exports = {
  apps: [
    {
      name: 'nerimity-cdn',
      script: './cdn_nerimity_go',
      interpreter: 'none',
      watch: false,
      time: true
    },
  ],
};